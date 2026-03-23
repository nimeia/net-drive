//go:build windows

package main

import (
	"log"

	"developer-mount/internal/winclientusergui"
)

func main() {
	if err := winclientusergui.Run(); err != nil {
		log.Fatal(err)
	}
}
