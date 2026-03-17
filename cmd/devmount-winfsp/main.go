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
	"developer-mount/internal/materialize"
	"developer-mount/internal/mountcore"
	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientdiag"
	"developer-mount/internal/winclientlog"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winfsp"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:17890", "server address")
	token := flag.String("token", "devmount-dev-token", "authentication token")
	clientInstanceID := flag.String("client-instance", "winfsp-smoke", "client instance id")
	op := flag.String("op", "volume", "operation: volume|getattr|readdir|read|mount|materialize|selfcheck|export-diagnostics")
	path := flag.String("path", "/", "mount-relative path")
	offset := flag.Int64("offset", 0, "read offset")
	length := flag.Uint("length", 64, "read length")
	maxEntries := flag.Uint("max-entries", 32, "max directory entries")
	mountPoint := flag.String("mount-point", "M:", "winfsp mount point (windows only for -op mount)")
	volumePrefix := flag.String("volume-prefix", "devmount", "winfsp volume prefix")
	localPath := flag.String("local-path", "devmount-local", "local target path for -op materialize")
	hostBackend := flag.String("host-backend", winclient.HostBackendAuto, "host backend: auto|preflight|dispatcher-v1")
	diagnosticsPath := flag.String("diagnostics-path", "devmount-diagnostics.zip", "output zip path for -op export-diagnostics")
	flag.Parse()

	cfg := winclient.Config{
		Addr:             *addr,
		Token:            *token,
		ClientInstanceID: *clientInstanceID,
		MountPoint:       *mountPoint,
		VolumePrefix:     *volumePrefix,
		Path:             *path,
		LocalPath:        *localPath,
		HostBackend:      *hostBackend,
		Offset:           *offset,
		Length:           uint32(*length),
		MaxEntries:       uint32(*maxEntries),
		LeaseSeconds:     30,
	}.Normalized()

	switch *op {
	case "selfcheck", "export-diagnostics":
		runDiagnostics(*op, cfg, *diagnosticsPath)
		return
	}

	cli := clientcore.New(cfg.Addr)
	if err := cli.Connect(); err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	if _, err := cli.Hello(); err != nil {
		log.Fatal(err)
	}
	if _, err := cli.Auth(cfg.Token); err != nil {
		log.Fatal(err)
	}
	if _, err := cli.CreateSession(cfg.ClientInstanceID, cfg.LeaseSeconds); err != nil {
		log.Fatal(err)
	}
	if _, err := cli.Heartbeat(); err != nil {
		log.Fatal(err)
	}
	mount := mountcore.New(cli, mountcore.Options{RootNodeID: cli.RootNodeID, VolumeName: cfg.VolumePrefix, ReadOnly: true})
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
		info, status := callbacks.GetFileInfo(cfg.Path)
		if status != winfsp.StatusSuccess {
			log.Fatalf("GetFileInfo(%s) status=0x%08x", cfg.Path, uint32(status))
		}
		fmt.Printf("getattr: path=%s node=%d dir=%v size=%d mode=%o\n", info.Path, info.NodeID, info.IsDirectory, info.Size, info.Mode)
	case "readdir":
		handle, status := callbacks.OpenDirectory(cfg.Path)
		if status != winfsp.StatusSuccess {
			log.Fatalf("OpenDirectory(%s) status=0x%08x", cfg.Path, uint32(status))
		}
		defer func() { _ = callbacks.Close(handle.HandleID) }()
		page, status := callbacks.ReadDirectory(handle.HandleID, 0, cfg.MaxEntries)
		if status != winfsp.StatusSuccess {
			log.Fatalf("ReadDirectory(%s) status=0x%08x", cfg.Path, uint32(status))
		}
		fmt.Printf("readdir: path=%s eof=%v next_cookie=%d entries=%d\n", handle.Info.Path, page.EOF, page.NextCookie, len(page.Entries))
		for _, entry := range page.Entries {
			fmt.Printf("- %s node=%d dir=%v size=%d\n", entry.Path, entry.NodeID, entry.IsDirectory, entry.Size)
		}
	case "read":
		handle, status := callbacks.Open(cfg.Path)
		if status != winfsp.StatusSuccess {
			log.Fatalf("Open(%s) status=0x%08x", cfg.Path, uint32(status))
		}
		defer func() { _ = callbacks.Close(handle.HandleID) }()
		data, eof, status := callbacks.Read(handle.HandleID, cfg.Offset, cfg.Length)
		if status != winfsp.StatusSuccess {
			log.Fatalf("Read(%s) status=0x%08x", cfg.Path, uint32(status))
		}
		fmt.Printf("read: path=%s offset=%d eof=%v bytes=%d\n", handle.Info.Path, cfg.Offset, eof, len(data))
		fmt.Printf("hex=%s\n", hex.EncodeToString(data))
		fmt.Printf("text=%q\n", string(data))
	case "materialize":
		stats, err := materialize.New(mount, cfg.Length, cfg.MaxEntries).Sync(context.Background(), cfg.Path, cfg.LocalPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("materialize: remote=%s local=%s directories=%d files=%d bytes=%d\n", cfg.Path, cfg.LocalPath, stats.Directories, stats.Files, stats.Bytes)
	case "mount":
		host := winfsp.NewHost(winfsp.HostConfig{MountPoint: cfg.MountPoint, VolumePrefix: cfg.VolumePrefix, Backend: cfg.HostBackend}, adapter)
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := host.Run(ctx); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported -op %q", *op)
	}
}

func runDiagnostics(op string, cfg winclient.Config, diagnosticsPath string) {
	logger, err := winclientlog.OpenDefault()
	if err != nil {
		log.Fatal(err)
	}
	tail, _ := logger.Tail(16 * 1024)
	report := winclientdiag.NewChecker().Run(context.Background(), cfg, winclientruntime.Snapshot{RequestedBackend: cfg.HostBackend}, "", logger.Path(), tail)
	if op == "selfcheck" {
		fmt.Print(report.Text())
		return
	}
	outputPath, err := winclientdiag.Export(diagnosticsPath, report)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("diagnostics exported to %s\n\n%s", outputPath, report.Text())
}
