package main

import (
	"flag"
	"log"

	"developer-mount/internal/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:17890", "listen address")
	root := flag.String("root", ".", "root directory exposed by the server")
	flag.Parse()

	srv := server.New(*addr)
	srv.RootPath = *root
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
