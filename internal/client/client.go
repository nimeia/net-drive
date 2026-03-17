package client

import "developer-mount/internal/clientcore"

type Client = clientcore.Client

func New(addr string) *Client {
	return clientcore.New(addr)
}

func decodeInto(payload []byte, out any) error {
	return clientcore.DecodeInto(payload, out)
}
