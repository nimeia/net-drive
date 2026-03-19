//go:build windows

package winfsp

import (
	"os"
	"strings"
	"sync"

	"developer-mount/internal/winclientlog"
)

var (
	nativeTraceOnce   sync.Once
	nativeTraceLogger *winclientlog.Logger
)

func nativeTraceInfo(code, message string, fields map[string]string) {
	nativeTrace(winclientlog.LevelInfo, code, message, fields)
}

func nativeTraceError(code, message string, fields map[string]string) {
	nativeTrace(winclientlog.LevelError, code, message, fields)
}

func nativeTrace(level winclientlog.Level, code, message string, fields map[string]string) {
	if strings.TrimSpace(os.Getenv("DEVMOUNT_WINFSP_TRACE")) == "0" {
		return
	}
	nativeTraceOnce.Do(func() {
		path, err := winclientlog.DefaultPath()
		if err != nil || strings.TrimSpace(path) == "" {
			return
		}
		logger := winclientlog.New(path)
		nativeTraceLogger = &logger
	})
	if nativeTraceLogger == nil {
		return
	}
	_ = nativeTraceLogger.Record(winclientlog.Entry{
		Level:     level,
		Code:      code,
		Component: "winfsp-native",
		Message:   message,
		Fields:    fields,
	})
}
