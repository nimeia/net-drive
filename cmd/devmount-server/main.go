package main

import (
	"developer-mount/internal/server"
	"flag"
	"log"
	"net/http"
)

func main() {
	configPath := flag.String("config", "", "optional server config JSON path")
	addr := flag.String("addr", "", "listen address override")
	root := flag.String("root", "", "root directory override")
	statusAddr := flag.String("status-addr", "", "optional status HTTP listen address")
	authToken := flag.String("auth-token", "", "authentication token override")
	auditLog := flag.String("audit-log", "", "audit log path override")
	flag.Parse()
	srv := server.New("127.0.0.1:17890")
	cfg := server.ServerConfig{}
	if *configPath != "" {
		loaded, err := server.LoadServerConfig(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		cfg = loaded
		if err := cfg.ApplyToServer(srv); err != nil {
			log.Fatal(err)
		}
	}
	if *addr != "" {
		srv.Addr = *addr
	}
	if *root != "" {
		srv.RootPath = *root
	}
	if *authToken != "" {
		srv.AuthToken = *authToken
	}
	if *auditLog != "" {
		audit, err := server.NewAuditLogger(*auditLog)
		if err != nil {
			log.Fatal(err)
		}
		srv.Audit = audit
	}
	if srv.Audit != nil {
		defer srv.Audit.Close()
	}
	listenStatus := *statusAddr
	if listenStatus == "" {
		listenStatus = cfg.StatusAddr
	}
	if listenStatus != "" {
		go func() {
			log.Printf("devmount status server listening on %s", listenStatus)
			if err := http.ListenAndServe(listenStatus, server.NewStatusHandler(srv)); err != nil {
				log.Printf("status server exited: %v", err)
			}
		}()
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
