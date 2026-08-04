package main

import (
	"fmt"
	"net/url"
)

// validateWorkerURL rejects control-plane URLs that would let an attacker MITM
// registration, updates, or the WebSocket connection. Plain HTTP is allowed
// only for loopback hosts during local development.
func validateWorkerURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid worker URL %q", raw)
	}

	if u.Scheme == "https" {
		return nil
	}

	if u.Scheme == "http" {
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("worker URL must use https, got %q", raw)
	}

	return fmt.Errorf("worker URL must use https, got %q", raw)
}
