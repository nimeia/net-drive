//go:build windows

package main

import (
	"developer-mount/internal/winclientgui"
	"log"
)

func main() {
	if err := winclientgui.Run(); err != nil {
		log.Fatal(err)
	}
}
