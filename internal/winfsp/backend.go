package winfsp

import "strings"

func normalizeRequestedBackend(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "auto":
		return "auto"
	case "preflight":
		return "preflight"
	case "dispatcher-v1":
		return "dispatcher-v1"
	default:
		return value
	}
}
