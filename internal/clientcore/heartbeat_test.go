package clientcore

import (
	"testing"
	"time"
)

func TestHeartbeatInterval(t *testing.T) {
	tests := []struct {
		name         string
		leaseSeconds uint32
		want         time.Duration
	}{
		{name: "default lease", leaseSeconds: 0, want: 10 * time.Second},
		{name: "thirty second lease", leaseSeconds: 30, want: 10 * time.Second},
		{name: "short lease clamps to minimum", leaseSeconds: 9, want: 5 * time.Second},
		{name: "long lease scales down", leaseSeconds: 120, want: 40 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := heartbeatInterval(tt.leaseSeconds); got != tt.want {
				t.Fatalf("heartbeatInterval(%d) = %s, want %s", tt.leaseSeconds, got, tt.want)
			}
		})
	}
}
