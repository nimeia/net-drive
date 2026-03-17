package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"developer-mount/internal/clientcore"
	"developer-mount/internal/mountcore"
	"developer-mount/internal/winfsp"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:17890", "server address")
	token := flag.String("token", "devmount-dev-token", "authentication token")
	clientInstanceID := flag.String("client-instance", "winfsp-smoke", "client instance id")
	op := flag.String("op", "volume", "operation: volume|getattr|readdir|read|mount")
	path := flag.String("path", "/", "mount-relative path")
	offset := flag.Int64("offset", 0, "read offset")
	length := flag.Uint("length", 64, "read length")
	maxEntries := flag.Uint("max-entries", 32, "max directory entries")
	mountPoint := flag.String("mount-point", "M:", "winfsp mount point (windows only for -op mount)")
	volumePrefix := flag.String("volume-prefix", "devmount", "winfsp volume prefix")
	flag.Parse()
	cli := clientcore.New(*addr)
	if err := cli.Connect(); err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	if _, err := cli.Hello(); err != nil {
		log.Fatal(err)
	}
	if _, err := cli.Auth(*token); err != nil {
		log.Fatal(err)
	}
	if _, err := cli.CreateSession(*clientInstanceID, 30); err != nil {
		log.Fatal(err)
	}
	if _, err := cli.Heartbeat(); err != nil {
		log.Fatal(err)
	}
	mount := mountcore.New(cli, mountcore.Options{RootNodeID: cli.RootNodeID, VolumeName: "devmount", ReadOnly: true})
	adapter := adapterpkg.New(mount)
	callbacks := winfsp.NewCallbacks(adapter)
	switch *op {
	case "volume":
		info, status := callbacks.GetVolumeInfo()
		if status != winfsp.StatusSuccess {
			log.Fatalf("GetVolumeInfo status=0x%08x", uint32(status))
		}
		fmt.Printf("volume: name=%s readonly=%v case_sensitive=%v max_component=%d\n", info.Name, info.ReadOnly, info.CaseSensitive, info.MaxComponentLength)
	case "getattr":
		info, status := callbacks.GetFileInfo(*path)
		if status != winfsp.StatusSuccess {
			log.Fatalf("GetFileInfo(%s) status=0x%08x", *path, uint32(status))
		}
		fmt.Printf("getattr: path=%s node=%d dir=%v size=%d mode=%o\n", info.Path, info.NodeID, info.IsDirectory, info.Size, info.Mode)
	case "readdir":
		handle, status := callbacks.OpenDirectory(*path)
		if status != winfsp.StatusSuccess {
			log.Fatalf("OpenDirectory(%s) status=0x%08x", *path, uint32(status))
		}
		defer func() { _ = callbacks.Close(handle.HandleID) }()
		page, status := callbacks.ReadDirectory(handle.HandleID, 0, uint32(*maxEntries))
		if status != winfsp.StatusSuccess {
			log.Fatalf("ReadDirectory(%s) status=0x%08x", *path, uint32(status))
		}
		fmt.Printf("readdir: path=%s eof=%v next_cookie=%d entries=%d\n", handle.Info.Path, page.EOF, page.NextCookie, len(page.Entries))
		for _, entry := range page.Entries {
			fmt.Printf("- %s node=%d dir=%v size=%d\n", entry.Path, entry.NodeID, entry.IsDirectory, entry.Size)
		}
	case "read":
		handle, status := callbacks.Open(*path)
		if status != winfsp.StatusSuccess {
			log.Fatalf("Open(%s) status=0x%08x", *path, uint32(status))
		}
		defer func() { _ = callbacks.Close(handle.HandleID) }()
		data, eof, status := callbacks.Read(handle.HandleID, *offset, uint32(*length))
		if status != winfsp.StatusSuccess {
			log.Fatalf("Read(%s) status=0x%08x", *path, uint32(status))
		}
		fmt.Printf("read: path=%s offset=%d eof=%v bytes=%d\n", handle.Info.Path, *offset, eof, len(data))
		fmt.Printf("hex=%s\n", hex.EncodeToString(data))
		fmt.Printf("text=%q\n", string(data))
	case "mount":
		host := winfsp.NewHost(winfsp.HostConfig{MountPoint: *mountPoint, VolumePrefix: *volumePrefix}, adapter)
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := host.Run(ctx); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported -op %q", *op)
	}
}
