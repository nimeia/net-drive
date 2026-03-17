package winclientlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const overrideEnvName = "DEVMOUNT_WINCLIENT_LOG_PATH"

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

func (l Logger) Path() string { return l.path }

func (l Logger) Append(level, message string) error {
	if strings.TrimSpace(l.path) == "" {
		return fmt.Errorf("log path is empty")
	}
	if l.now == nil {
		l.now = time.Now
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
	line := fmt.Sprintf("%s [%s] %s\n", l.now().Format(time.RFC3339), strings.ToUpper(strings.TrimSpace(level)), strings.TrimSpace(message))
	_, err = f.WriteString(line)
	return err
}

func (l Logger) Info(message string) error  { return l.Append("info", message) }
func (l Logger) Error(message string) error { return l.Append("error", message) }

func (l Logger) Tail(maxBytes int) (string, error) {
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
