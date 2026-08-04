package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	// 1. Setup mock IMDS server
	imdsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			w.Write([]byte("mock-token"))
			return
		}
		if r.URL.Path == "/latest/dynamic/instance-identity/pkcs7" {
			w.Write([]byte("mock-pkcs7-signature"))
			return
		}
		if r.URL.Path == "/latest/dynamic/instance-identity/signature" {
			w.Write([]byte("mock-doc-signature"))
			return
		}
		if r.URL.Path == "/latest/dynamic/instance-identity/document" {
			w.Write([]byte(`{"instanceId":"i-12345"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer imdsServer.Close()

	originalURL := imdsBaseURL
	imdsBaseURL = imdsServer.URL
	defer func() { imdsBaseURL = originalURL }()

	// 2. Setup mock API Server (Cloudflare Worker)
	var receivedPayload map[string]string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agent/register" && r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedPayload)

			// Return success with JWT
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jwt": "mock-jwt-token-xyz"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiServer.Close()

	// 3. Execute
	config := &Config{
		ServerID:          "srv_test_abc",
		RegistrationToken: "tok_test_123",
		WorkerURL:         apiServer.URL,
	}
	identity := &InstanceIdentity{InstanceID: "i-12345"}

	jwt, err := Register(context.Background(), config, identity)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 4. Assertions
	if jwt != "mock-jwt-token-xyz" {
		t.Errorf("Expected JWT to be mock-jwt-token-xyz, got %s", jwt)
	}

	if receivedPayload["serverId"] != "srv_test_abc" {
		t.Errorf("Expected serverId in payload to be srv_test_abc, got %s", receivedPayload["serverId"])
	}
	if receivedPayload["token"] != "tok_test_123" {
		t.Errorf("Expected token in payload to be tok_test_123, got %s", receivedPayload["token"])
	}
	if receivedPayload["identityDocument"] != `{"instanceId":"i-12345"}` {
		t.Errorf("Expected identityDocument to be passed correctly, got %s", receivedPayload["identityDocument"])
	}
	if receivedPayload["pkcs7Signature"] != "mock-pkcs7-signature" {
		t.Errorf("Expected pkcs7Signature to be passed correctly, got %s", receivedPayload["pkcs7Signature"])
	}
	if receivedPayload["documentSignature"] != "mock-doc-signature" {
		t.Errorf("Expected documentSignature to be passed correctly, got %s", receivedPayload["documentSignature"])
	}
}

func TestRegister_ApiFailure(t *testing.T) {
	// 1. Setup mock IMDS server
	imdsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("mock"))
	}))
	defer imdsServer.Close()

	originalURL := imdsBaseURL
	imdsBaseURL = imdsServer.URL
	defer func() { imdsBaseURL = originalURL }()

	// 2. Setup mock API Server that FAILS
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer apiServer.Close()

	config := &Config{
		ServerID:          "srv_test",
		RegistrationToken: "tok_test",
		WorkerURL:         apiServer.URL,
	}

	_, err := Register(context.Background(), config, nil)
	if err == nil {
		t.Fatalf("Expected error when API fails, got nil")
	}
}
