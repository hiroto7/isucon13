package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

var dnsAPIClient = &http.Client{Timeout: 2 * time.Second}

func purgeDNSName(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://127.0.0.1:8081/api/v1/servers/localhost/cache/flush?domain="+url.QueryEscape(name+"."), nil)
	if err != nil {
		return err
	}
	key := os.Getenv("ISUCON13_POWERDNS_API_KEY")
	if key == "" {
		key = "isudns"
	}
	req.Header.Set("X-API-Key", key)
	resp, err := dnsAPIClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DNS cache flush status %d", resp.StatusCode)
	}
	return nil
}
