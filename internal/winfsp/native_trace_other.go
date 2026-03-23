//go:build !windows

package winfsp

func nativeTraceError(code, message string, fields map[string]string) {}
