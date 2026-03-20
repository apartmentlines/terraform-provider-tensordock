package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSecretValueReturnsLiveSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/secrets/secret-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"secret-1","name":"deploy-key","type":"SSHKEY","value":"ssh-ed25519 AAAA-from-secret"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	resource := &SecretValueEphemeralResource{client: client}
	result, err := resource.fetchSecretValue(context.Background(), "secret-1")
	if err != nil {
		t.Fatalf("fetchSecretValue returned error: %v", err)
	}

	if result.SecretID.ValueString() != "secret-1" {
		t.Fatalf("unexpected secret_id: %q", result.SecretID.ValueString())
	}
	if result.ID.ValueString() != "secret-1" {
		t.Fatalf("unexpected id: %q", result.ID.ValueString())
	}
	if result.Name.ValueString() != "deploy-key" {
		t.Fatalf("unexpected name: %q", result.Name.ValueString())
	}
	if result.Type.ValueString() != "SSHKEY" {
		t.Fatalf("unexpected type: %q", result.Type.ValueString())
	}
	if result.Value.ValueString() != "ssh-ed25519 AAAA-from-secret" {
		t.Fatalf("unexpected value: %q", result.Value.ValueString())
	}
}

func TestFetchSecretValueRejectsEmptyValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"secret-1","name":"deploy-key","type":"SSHKEY","value":""}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	resource := &SecretValueEphemeralResource{client: client}
	if _, err := resource.fetchSecretValue(context.Background(), "secret-1"); err == nil {
		t.Fatal("expected fetchSecretValue to reject empty secret values")
	}
}
