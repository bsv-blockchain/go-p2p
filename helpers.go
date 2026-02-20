package p2p

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

// GetPublicIP fetches the public IP address from ifconfig.me
func GetPublicIP(ctx context.Context) (string, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			// Force the use of IPv4 by specifying 'tcp4' as the network
			return (&net.Dialer{}).DialContext(ctx, "tcp4", addr)
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://ifconfig.me/ip", nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req) //nolint:gosec // G704: URL is hardcoded, not user-supplied
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), resp.Body.Close()
}
