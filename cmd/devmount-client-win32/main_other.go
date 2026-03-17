//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "devmount-client-win32 is available on Windows only")
	os.Exit(1)
}
