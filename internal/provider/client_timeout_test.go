package provider

import (
	"testing"
)

func TestNewClientUsesExtendedDefaultTimeout(t *testing.T) {
	client, err := NewClient("https://example.com/api/v2", "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if client.httpClient == nil {
		t.Fatal("expected http client to be initialized")
	}
	if client.httpClient.Timeout != defaultHTTPTimeout {
		t.Fatalf("unexpected timeout: got %v want %v", client.httpClient.Timeout, defaultHTTPTimeout)
	}
}
