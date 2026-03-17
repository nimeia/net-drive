package materialize

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"developer-mount/internal/mountcore"
)

type Stats struct {
	Directories int
	Files       int
	Bytes       int64
}

type Materializer struct {
	mount        *mountcore.Mount
	readLength   uint32
	maxEntries   uint32
	preserveTime bool
}

func New(mount *mountcore.Mount, readLength uint32, maxEntries uint32) *Materializer {
	if readLength == 0 {
		readLength = 1 << 20
	}
	if maxEntries == 0 {
		maxEntries = 128
	}
	return &Materializer{mount: mount, readLength: readLength, maxEntries: maxEntries, preserveTime: true}
}

func (m *Materializer) Sync(ctx context.Context, remotePath string, localPath string) (Stats, error) {
	if strings.TrimSpace(localPath) == "" {
		return Stats{}, fmt.Errorf("local path is required")
	}
	if err := ctx.Err(); err != nil {
		return Stats{}, err
	}
	info, err := m.mount.GetAttr(remotePath)
	if err != nil {
		return Stats{}, err
	}
	if info.IsDirectory {
		return m.syncDirectory(ctx, info.Path, localPath)
	}
	return m.syncFile(ctx, info.Path, localPath, info)
}

func (m *Materializer) syncDirectory(ctx context.Context, remotePath string, localPath string) (Stats, error) {
	if err := ctx.Err(); err != nil { return Stats{}, err }
	if err := os.MkdirAll(localPath, 0o755); err != nil { return Stats{}, err }
	handle, err := m.mount.OpenDirectory(remotePath)
	if err != nil { return Stats{}, err }
	defer func() { _ = m.mount.Close(handle.HandleID) }()
	stats := Stats{Directories: 1}
	var cookie uint64
	for {
		if err := ctx.Err(); err != nil { return stats, err }
		page, err := m.mount.ReadDirectory(handle.HandleID, cookie, m.maxEntries)
		if err != nil { return stats, err }
		for _, entry := range page.Entries {
			name, err := validateLocalName(entry.Name)
			if err != nil { return stats, fmt.Errorf("%s: %w", entry.Path, err) }
			childLocalPath := filepath.Join(localPath, name)
			var childStats Stats
			if entry.IsDirectory { childStats, err = m.syncDirectory(ctx, entry.Path, childLocalPath) } else { childStats, err = m.syncFile(ctx, entry.Path, childLocalPath, entry) }
			if err != nil { return stats, err }
			stats.Directories += childStats.Directories
			stats.Files += childStats.Files
			stats.Bytes += childStats.Bytes
		}
		if page.EOF { break }
		cookie = page.NextCookie
	}
	return stats, nil
}

func (m *Materializer) syncFile(ctx context.Context, remotePath string, localPath string, info mountcore.FileInfo) (Stats, error) {
	if err := ctx.Err(); err != nil { return Stats{}, err }
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil { return Stats{}, err }
	handle, err := m.mount.Open(remotePath)
	if err != nil { return Stats{}, err }
	defer func() { _ = m.mount.Close(handle.HandleID) }()
	file, err := os.Create(localPath)
	if err != nil { return Stats{}, err }
	defer file.Close()
	stats := Stats{Files: 1}
	var offset int64
	for {
		if err := ctx.Err(); err != nil { return stats, err }
		result, err := m.mount.Read(handle.HandleID, offset, m.readLength)
		if err != nil { return stats, err }
		if len(result.Data) > 0 {
			n, err := file.Write(result.Data)
			if err != nil { return stats, err }
			stats.Bytes += int64(n)
			offset += int64(n)
		}
		if result.EOF || len(result.Data) == 0 { break }
	}
	if err := file.Close(); err != nil { return stats, err }
	if m.preserveTime {
		if modTime, err := parseModTime(info.ModTime); err == nil && !modTime.IsZero() { _ = os.Chtimes(localPath, modTime, modTime) }
	}
	return stats, nil
}

func validateLocalName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" { return "", fmt.Errorf("empty entry name") }
	if name == "." || name == ".." { return "", fmt.Errorf("invalid entry name %q", name) }
	if strings.Contains(name, "/") || strings.Contains(name, `\\`) { return "", fmt.Errorf("entry name %q contains path separator", name) }
	return name, nil
}

func parseModTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" { return time.Time{}, nil }
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"}
	for _, layout := range layouts { if t, err := time.Parse(layout, value); err == nil { return t, nil } }
	return time.Time{}, fmt.Errorf("unsupported mod time %q", value)
}
