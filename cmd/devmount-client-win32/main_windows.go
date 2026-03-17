//go:build windows

package main

import (
	"log"

	"developer-mount/internal/winclientgui"
)

func main() {
	if err := winclientgui.Run(); err != nil {
		log.Fatal(err)
	}
}
