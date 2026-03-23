//go:build !windows

package winclientusergui

import "fmt"

func Run() error { return fmt.Errorf("windows only") }
