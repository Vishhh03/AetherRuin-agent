package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func Register(ctx context.Context, config *Config, identity *InstanceIdentity) (string, error) {
	token, err := getIMDSv2Token(ctx)
	if err != nil {
		return "", fmt.Errorf("IMDSv2 token: %w", err)
	}

	pkcs7Bytes, err := fetchWithToken(ctx, imdsBaseURL+"/latest/dynamic/instance-identity/pkcs7", token)
	if err != nil {
		return "", fmt.Errorf("PKCS7: %w", err)
	}
	pkcs7 := string(pkcs7Bytes)

	signatureBytes, err := fetchWithToken(ctx, imdsBaseURL+"/latest/dynamic/instance-identity/signature", token)
	if err != nil {
		return "", fmt.Errorf("signature: %w", err)
	}
	documentSignature := string(signatureBytes)

	docBytes, err := fetchWithToken(ctx, imdsBaseURL+"/latest/dynamic/instance-identity/document", token)
	if err != nil {
		return "", fmt.Errorf("identity doc: %w", err)
	}
	identityDoc := string(docBytes)

	payload := map[string]string{
		"serverId":          config.ServerID,
		"token":             config.RegistrationToken,
		"identityDocument":  identityDoc,
		"pkcs7Signature":    pkcs7,
		"documentSignature": documentSignature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(config.WorkerURL, "/") + "/agent/register"
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AetherRuin-Agent/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("register request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("register returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		JWT                string `json:"jwt"`
		RegistrationToken  string `json:"registrationToken"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse register response: %w", err)
	}

	if result.JWT == "" {
		return "", fmt.Errorf("no JWT in register response")
	}

	if result.RegistrationToken != "" {
		config.RegistrationToken = result.RegistrationToken
	}

	log.Println("Registration successful")
	return result.JWT, nil
}

func GetUptimeSeconds() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return 0
	}
	var uptime float64
	fmt.Sscanf(parts[0], "%f", &uptime)
	return int64(uptime)
}
