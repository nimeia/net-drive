package main

import (
	"flag"
	"fmt"
	"log"

	"developer-mount/internal/client"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:17890", "server address")
	token := flag.String("token", "devmount-dev-token", "authentication token")
	flag.Parse()

	c := client.New(*addr)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	helloResp, err := c.Hello()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("hello: server=%s version=%s selected=%d\n", helloResp.ServerName, helloResp.ServerVersion, helloResp.SelectedProtocolVersion)

	authResp, err := c.Auth(*token)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("auth: principal=%s authenticated=%v\n", authResp.PrincipalID, authResp.Authenticated)

	sessionResp, err := c.CreateSession("client-1", 30)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("session: id=%d state=%s expires=%s\n", sessionResp.SessionID, sessionResp.State, sessionResp.ExpiresAt)

	hbResp, err := c.Heartbeat()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("heartbeat: state=%s expires=%s\n", hbResp.State, hbResp.ExpiresAt)

	rootAttr, err := c.GetAttr(c.RootNodeID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("root: node=%d type=%s\n", rootAttr.Entry.NodeID, rootAttr.Entry.FileType)

	dirResp, err := c.OpenDir(c.RootNodeID)
	if err != nil {
		log.Fatal(err)
	}
	listResp, err := c.ReadDir(dirResp.DirCursorID, 0, 16)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := c.CloseDir(dirResp.DirCursorID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("root entries (%d):\n", len(listResp.Entries))
	for _, entry := range listResp.Entries {
		fmt.Printf("- %s [%s]\n", entry.Name, entry.FileType)
	}
}
