package main

import (
	"flag"
	"log"

	"developer-mount/internal/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:17890", "listen address")
	flag.Parse()

	srv := server.New(*addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
