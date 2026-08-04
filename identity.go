package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var imdsBaseURL = "http://169.254.169.254"

type InstanceIdentity struct {
	InstanceID   string `json:"instanceId"`
	Region       string `json:"region"`
	AccountID    string `json:"accountId"`
	ImageID      string `json:"imageId"`
	InstanceType string `json:"instanceType"`
}

func FetchInstanceIdentity(ctx context.Context) (*InstanceIdentity, error) {
	// Step 1: Get IMDSv2 token
	token, err := getIMDSv2Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("IMDSv2 token: %w", err)
	}

	// Step 2: Fetch identity document
	doc, err := fetchWithToken(ctx, imdsBaseURL+"/latest/dynamic/instance-identity/document", token)
	if err != nil {
		return nil, fmt.Errorf("identity document: %w", err)
	}

	var identity InstanceIdentity
	if err := json.Unmarshal(doc, &identity); err != nil {
		return nil, fmt.Errorf("parse identity: %w", err)
	}

	return &identity, nil
}

func getIMDSv2Token(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", imdsBaseURL+"/latest/api/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("IMDSv2 token request returned %d", resp.StatusCode)
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func fetchWithToken(ctx context.Context, url, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-aws-ec2-metadata-token", token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request to %s returned %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// GetPublicIP fetches the instance public IP from metadata
func GetPublicIP() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	token, err := getIMDSv2Token(ctx)
	if err != nil {
		return "", err
	}

	ip, err := fetchWithToken(ctx, imdsBaseURL+"/latest/meta-data/public-ipv4", token)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}
