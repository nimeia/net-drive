package winclient

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"developer-mount/internal/clientcore"
	"developer-mount/internal/materialize"
	"developer-mount/internal/mountcore"
)

type Runner struct{}

func NewRunner() Runner {
	return Runner{}
}

func (Runner) Execute(ctx context.Context, config Config, op Operation) (string, error) {
	config = config.Normalized()
	if err := config.Validate(op); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	cli := clientcore.New(config.Addr)
	if err := cli.Connect(); err != nil {
		return "", err
	}
	defer cli.Close()

	helloResp, err := cli.Hello()
	if err != nil {
		return "", err
	}
	authResp, err := cli.Auth(config.Token)
	if err != nil {
		return "", err
	}
	sessionResp, err := cli.CreateSession(config.ClientInstanceID, config.LeaseSeconds)
	if err != nil {
		return "", err
	}
	hbResp, err := cli.Heartbeat()
	if err != nil {
		return "", err
	}

	mount := mountcore.New(cli, mountcore.Options{
		RootNodeID: cli.RootNodeID,
		VolumeName: config.VolumePrefix,
		ReadOnly:   true,
	})

	var b strings.Builder
	fmt.Fprintf(&b, "hello: server=%s version=%s selected=%d\n", helloResp.ServerName, helloResp.ServerVersion, helloResp.SelectedProtocolVersion)
	fmt.Fprintf(&b, "auth: principal=%s authenticated=%v\n", authResp.PrincipalID, authResp.Authenticated)
	fmt.Fprintf(&b, "session: id=%d state=%s expires=%s\n", sessionResp.SessionID, sessionResp.State, sessionResp.ExpiresAt)
	fmt.Fprintf(&b, "heartbeat: state=%s expires=%s\n", hbResp.State, hbResp.ExpiresAt)

	switch op {
	case OperationVolume:
		info := mount.VolumeInfo()
		fmt.Fprintf(&b, "volume: name=%s readonly=%v case_sensitive=%v max_component=%d\n", info.Name, info.ReadOnly, info.CaseSensitive, info.MaxComponentLength)
	case OperationGetAttr:
		info, err := mount.GetAttr(config.Path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "getattr: path=%s node=%d dir=%v size=%d mode=%o\n", info.Path, info.NodeID, info.IsDirectory, info.Size, info.Mode)
	case OperationReadDir:
		handle, err := mount.OpenDirectory(config.Path)
		if err != nil {
			return "", err
		}
		defer func() { _ = mount.Close(handle.HandleID) }()
		page, err := mount.ReadDirectory(handle.HandleID, 0, config.MaxEntries)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "readdir: path=%s eof=%v next_cookie=%d entries=%d\n", handle.Info.Path, page.EOF, page.NextCookie, len(page.Entries))
		for _, entry := range page.Entries {
			fmt.Fprintf(&b, "- %s node=%d dir=%v size=%d\n", entry.Path, entry.NodeID, entry.IsDirectory, entry.Size)
		}
	case OperationRead:
		handle, err := mount.Open(config.Path)
		if err != nil {
			return "", err
		}
		defer func() { _ = mount.Close(handle.HandleID) }()
		result, err := mount.Read(handle.HandleID, config.Offset, config.Length)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "read: path=%s offset=%d eof=%v bytes=%d\n", handle.Info.Path, config.Offset, result.EOF, len(result.Data))
		fmt.Fprintf(&b, "hex=%s\n", hex.EncodeToString(result.Data))
		fmt.Fprintf(&b, "text=%q\n", string(result.Data))
	case OperationMaterialize:
		stats, err := materialize.New(mount, config.Length, config.MaxEntries).Sync(ctx, config.Path, config.LocalPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "materialize: remote=%s local=%s directories=%d files=%d bytes=%d\n", config.Path, config.LocalPath, stats.Directories, stats.Files, stats.Bytes)
	default:
		return "", fmt.Errorf("unsupported operation %q", op)
	}

	return b.String(), nil
}
