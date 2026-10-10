// Package healthcheck implements the server's "healthcheck" subcommand: the
// runtime image has no shell or curl, so Docker's HEALTHCHECK runs the binary.
package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Run requests /health on localhost:port and returns an error unless it answers 200.
func Run(ctx context.Context, port string) error {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	url := "http://" + net.JoinHostPort("127.0.0.1", port) + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("health request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health: HTTP %d", resp.StatusCode)
	}
	return nil
}
