package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func archSuffix() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return "amd64"
}

// validateDownloadHTTPS rejects plain-HTTP and non-http(s) download URLs so a
// downgrade or MITM cannot install a malicious binary.
func validateDownloadHTTPS(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid download URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("download URL must use https, got %q", raw)
	}
	return nil
}

type UpdateInfo struct {
	Version     string `json:"latestVersion"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
}

func (a *Agent) StartUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Initial check after 5 minutes (not immediate, let agent stabilize)
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Minute):
		a.checkForUpdate(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.checkForUpdate(ctx)
		}
	}
}

func (a *Agent) checkForUpdate(ctx context.Context) {
	client := &http.Client{Timeout: 10 * time.Second}

	versionURL, err := url.Parse(strings.TrimRight(a.config.WorkerURL, "/") + "/agent/version")
	if err != nil {
		return
	}
	query := versionURL.Query()
	query.Set("serverId", a.config.ServerID)
	query.Set("currentVersion", Version)
	versionURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", versionURL.String(), nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var info UpdateInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return
	}

	if info.Version == Version {
		return // already up to date
	}

	log.Printf("Update available: %s -> %s", Version, info.Version)
	a.performUpdate(ctx, &info)
}

func (a *Agent) performUpdate(ctx context.Context, info *UpdateInfo) {
	// A missing SHA256 is never acceptable: it would allow a tampered or MITM'd
	// binary to be installed and executed as root.
	if info.SHA256 == "" {
		log.Println("Update rejected: no SHA256 provided")
		return
	}

	log.Printf("Downloading update %s...", info.Version)

	// Download new binary
	newBinary := "/tmp/minecraft-agent-new"
	downloadURL := info.DownloadURL
	if downloadURL == "" {
		downloadURL = a.config.WorkerURL + "/agent/download?arch=" + archSuffix()
	}
	if err := validateDownloadHTTPS(downloadURL); err != nil {
		log.Printf("Update rejected: %v", err)
		return
	}
	if err := downloadFile(ctx, downloadURL, newBinary); err != nil {
		log.Printf("Download failed: %v", err)
		return
	}

	// SHA256 is mandatory (checked above); verify before replacing the binary.
	if !verifySHA256(newBinary, info.SHA256) {
		log.Println("SHA256 mismatch, aborting update")
		os.Remove(newBinary)
		return
	}

	// Make executable
	if err := os.Chmod(newBinary, 0755); err != nil {
		log.Printf("chmod failed: %v", err)
		return
	}

	// Move current binary to .prev
	currentBinary := "/usr/local/bin/minecraft-agent"
	prevBinary := currentBinary + ".prev"

	if err := os.Rename(currentBinary, prevBinary); err != nil {
		log.Printf("rename current failed: %v", err)
		return
	}

	// Move new binary to install path
	if err := os.Rename(newBinary, currentBinary); err != nil {
		log.Printf("rename new failed: %v, rolling back", err)
		os.Rename(prevBinary, currentBinary)
		return
	}

	log.Printf("Updated to %s, restarting", info.Version)

	// Restart via systemctl
	cmd := exec.Command("systemctl", "restart", "minecraft-agent")
	if err := cmd.Run(); err != nil {
		log.Printf("systemctl restart failed: %v", err)
		// Rollback
		os.Rename(prevBinary, currentBinary)
	}
}

func downloadFile(ctx context.Context, url, dest string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func verifySHA256(filePath, expected string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}

	return hex.EncodeToString(hash.Sum(nil)) == expected
}
