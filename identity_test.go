package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchInstanceIdentity(t *testing.T) {
	// Setup mock IMDS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" && r.Method == "PUT" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock-token-123"))
			return
		}
		if r.URL.Path == "/latest/dynamic/instance-identity/document" && r.Method == "GET" {
			if r.Header.Get("X-aws-ec2-metadata-token") != "mock-token-123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"instanceId": "i-1234567890abcdef0",
				"region": "us-east-1",
				"accountId": "123456789012",
				"imageId": "ami-0abcdef1234567890",
				"instanceType": "t4g.small"
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Override the IMDS base URL for the test
	originalURL := imdsBaseURL
	imdsBaseURL = server.URL
	defer func() { imdsBaseURL = originalURL }()

	// Execute
	identity, err := FetchInstanceIdentity(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assertions
	if identity.InstanceID != "i-1234567890abcdef0" {
		t.Errorf("Expected instanceId to be i-1234567890abcdef0, got %s", identity.InstanceID)
	}
	if identity.Region != "us-east-1" {
		t.Errorf("Expected region to be us-east-1, got %s", identity.Region)
	}
}

func TestGetPublicIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock-token"))
			return
		}
		if r.URL.Path == "/latest/meta-data/public-ipv4" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("203.0.113.50"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalURL := imdsBaseURL
	imdsBaseURL = server.URL
	defer func() { imdsBaseURL = originalURL }()

	ip, err := GetPublicIP()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if ip != "203.0.113.50" {
		t.Errorf("Expected IP to be 203.0.113.50, got %s", ip)
	}
}

func TestFetchInstanceIdentity_TokenFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	originalURL := imdsBaseURL
	imdsBaseURL = server.URL
	defer func() { imdsBaseURL = originalURL }()

	_, err := FetchInstanceIdentity(context.Background())
	if err == nil {
		t.Fatalf("Expected error when token fetch fails, got nil")
	}
}
