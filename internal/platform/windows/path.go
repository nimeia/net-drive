package windows

import (
	"fmt"
	gopath "path"
	"strings"
)

// NormalizeMountPath converts a WinFsp-style path into a stable absolute mount
// path rooted at "/". Parent traversal outside the mount root is rejected.
func NormalizeMountPath(input string) (string, error) {
	replaced := strings.ReplaceAll(strings.TrimSpace(input), "\\", "/")
	if replaced == "" || replaced == "." {
		return "/", nil
	}
	trimmed := strings.TrimPrefix(replaced, "/")
	parts := strings.Split(trimmed, "/")
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			return "", fmt.Errorf("invalid path %q: parent traversal is not allowed", input)
		default:
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "/", nil
	}
	return "/" + gopath.Clean(strings.Join(cleanParts, "/")), nil
}

func SplitMountPath(input string) ([]string, error) {
	normalized, err := NormalizeMountPath(input)
	if err != nil {
		return nil, err
	}
	if normalized == "/" {
		return nil, nil
	}
	return strings.Split(strings.TrimPrefix(normalized, "/"), "/"), nil
}

func JoinMountPath(parent, name string) string {
	normalizedParent, err := NormalizeMountPath(parent)
	if err != nil {
		normalizedParent = "/"
	}
	child := strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	child = strings.Trim(child, "/")
	joined := normalizedParent
	if child != "" {
		if normalizedParent == "/" {
			joined = "/" + child
		} else {
			joined = normalizedParent + "/" + child
		}
	}
	normalized, err := NormalizeMountPath(joined)
	if err != nil {
		return joined
	}
	return normalized
}

func IsRootPath(input string) bool {
	normalized, err := NormalizeMountPath(input)
	if err != nil {
		return false
	}
	return normalized == "/"
}
