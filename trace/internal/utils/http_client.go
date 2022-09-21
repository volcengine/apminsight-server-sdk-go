package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	unixNetwork = "unix"
)

// NewHTTPClientViaUDS return a http.client based on http
func NewHTTPClientViaUDS(sock string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, unixNetwork, sock)
			},
		},
		Timeout: timeout,
	}
}

func URLViaUDS(path string) string {
	return fmt.Sprintf("http://%s/%s", unixNetwork, strings.TrimPrefix(path, "/"))
}
