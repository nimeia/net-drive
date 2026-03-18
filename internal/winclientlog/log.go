package winclientlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const overrideEnvName = "DEVMOUNT_WINCLIENT_LOG_PATH"

type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Entry struct {
	Timestamp time.Time
	Level     Level
	Code      string
	Component string
	Message   string
	Fields    map[string]string
}

type Logger struct {
	path string
	mu   sync.Mutex
	now  func() time.Time
}

func New(path string) Logger { return Logger{path: path, now: time.Now} }
func OpenDefault() (Logger, error) {
	path, err := DefaultPath()
	if err != nil {
		return Logger{}, err
	}
	return New(path), nil
}
func DefaultPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(overrideEnvName)); override != "" {
		return override, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "developer-mount", "logs", "win32-client.log"), nil
}
func (l *Logger) Path() string { return l.path }
func (l *Logger) Record(entry Entry) error {
	if strings.TrimSpace(l.path) == "" {
		return fmt.Errorf("log path is empty")
	}
	if l.now == nil {
		l.now = time.Now
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = l.now()
	}
	if entry.Level == "" {
		entry.Level = LevelInfo
	}
	if strings.TrimSpace(entry.Code) == "" {
		entry.Code = "generic"
	}
	if strings.TrimSpace(entry.Component) == "" {
		entry.Component = "win32-client"
	}
	entry.Message = strings.TrimSpace(entry.Message)
	if entry.Message == "" {
		entry.Message = "(empty message)"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(formatEntry(entry))
	return err
}
func (l *Logger) Append(level, message string) error {
	return l.Record(Entry{Level: Level(strings.ToLower(strings.TrimSpace(level))), Message: message})
}
func (l *Logger) Info(message string) error {
	return l.Record(Entry{Level: LevelInfo, Code: "ui.info", Component: "win32-client", Message: message})
}
func (l *Logger) Warn(message string) error {
	return l.Record(Entry{Level: LevelWarn, Code: "ui.warn", Component: "win32-client", Message: message})
}
func (l *Logger) Error(message string) error {
	return l.Record(Entry{Level: LevelError, Code: "ui.error", Component: "win32-client", Message: message})
}
func (l *Logger) Tail(maxBytes int) (string, error) {
	if strings.TrimSpace(l.path) == "" {
		return "", fmt.Errorf("log path is empty")
	}
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read log file: %w", err)
	}
	if maxBytes > 0 && len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data), nil
}
func formatEntry(entry Entry) string {
	parts := []string{fmt.Sprintf("ts=%s", entry.Timestamp.UTC().Format(time.RFC3339)), fmt.Sprintf("level=%s", strings.ToUpper(string(entry.Level))), fmt.Sprintf("code=%s", quoteField(entry.Code)), fmt.Sprintf("component=%s", quoteField(entry.Component)), fmt.Sprintf("msg=%s", quoteField(entry.Message))}
	if len(entry.Fields) > 0 {
		keys := make([]string, 0, len(entry.Fields))
		for key := range entry.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", sanitizeKey(key), quoteField(entry.Fields[key])))
		}
	}
	return strings.Join(parts, " ") + "\n"
}
func sanitizeKey(key string) string {
	key = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(key, " ", "_"), "\t", "_"))
	if key == "" {
		return "field"
	}
	return key
}
func quoteField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return `""`
	}
	escaped := strings.ReplaceAll(value, `\\`, `\\\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\\"`)
	return `"` + escaped + `"`
}
