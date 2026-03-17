package server

import (
	"os"
	"path/filepath"
	"strings"

	"developer-mount/internal/protocol"
)

type prefetchPriority int

const (
	prefetchPriorityNormal prefetchPriority = iota
	prefetchPriorityHigh
)

type prefetchJobKind string

const (
	prefetchJobDirSnapshot prefetchJobKind = "dir-snapshot"
	prefetchJobSmallFile   prefetchJobKind = "small-file"
)

type prefetchJob struct {
	priority prefetchPriority
	kind     prefetchJobKind
	nodeID   uint64
	relPath  string
}

type workspaceProfile struct {
	HotDirs            map[string]struct{}
	HotFiles           map[string]struct{}
	IgnoreDirs         map[string]struct{}
	SmallFileThreshold int64
}

func defaultWorkspaceProfile() workspaceProfile {
	return workspaceProfile{
		HotDirs: map[string]struct{}{
			".git": {}, ".vscode": {}, "src": {}, "cmd": {}, "internal": {}, "pkg": {}, "include": {}, "lib": {},
		},
		HotFiles: map[string]struct{}{
			"package.json": {}, "tsconfig.json": {}, "go.mod": {}, "Cargo.toml": {}, "pyproject.toml": {},
			".gitignore": {}, "README.md": {}, "README": {},
		},
		IgnoreDirs: map[string]struct{}{
			"node_modules": {}, "dist": {}, "build": {}, "target": {}, ".venv": {}, "__pycache__": {},
		},
		SmallFileThreshold: 256 * 1024,
	}
}

func inferWorkspaceProfile(rootPath string) workspaceProfile {
	profile := defaultWorkspaceProfile()
	candidates := []string{"web", "app", "apps", "services", "configs", "config", "scripts", "test", "tests"}
	for _, name := range candidates {
		if info, err := os.Stat(filepath.Join(rootPath, name)); err == nil && info.IsDir() {
			profile.HotDirs[name] = struct{}{}
		}
	}
	fileCandidates := []string{"pnpm-lock.yaml", "yarn.lock", "package-lock.json", "requirements.txt", "Makefile", "Taskfile.yml"}
	for _, name := range fileCandidates {
		if info, err := os.Stat(filepath.Join(rootPath, name)); err == nil && !info.IsDir() {
			profile.HotFiles[name] = struct{}{}
		}
	}
	return profile
}

func (p workspaceProfile) IsHotDir(name string) bool {
	_, ok := p.HotDirs[name]
	return ok
}

func (p workspaceProfile) IsIgnoredDir(name string) bool {
	_, ok := p.IgnoreDirs[name]
	return ok
}

func (p workspaceProfile) IsHotFile(name string) bool {
	_, ok := p.HotFiles[name]
	return ok
}

func (p workspaceProfile) IsSmallFileCandidate(relPath string, info protocol.NodeInfo) bool {
	if info.FileType != protocol.FileTypeFile {
		return false
	}
	if info.Size > p.SmallFileThreshold {
		return false
	}
	base := filepath.Base(relPath)
	if p.IsHotFile(base) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".json", ".ts", ".tsx", ".js", ".jsx", ".py", ".go", ".c", ".cc", ".cpp", ".h", ".hpp", ".rs", ".md", ".toml", ".yml", ".yaml", ".txt", ".lock":
		return true
	default:
		return false
	}
}
