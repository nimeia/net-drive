//go:build windows

package winclientusergui

import "developer-mount/internal/winclientgui"

// Run currently reuses the existing Win32 shell while the final-user page set is split incrementally.
func Run() error {
	return winclientgui.Run()
}
