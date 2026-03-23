//go:build windows

package main

import (
	"developer-mount/internal/winclientusergui"
	"log"
)

func main() {
	if err := winclientusergui.Run(); err != nil {
		log.Fatal(err)
	}
}
