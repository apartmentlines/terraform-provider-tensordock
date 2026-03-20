package provider

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestDoJSONEmitsRequestAndResponseLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"secret-1"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)

	_, err = client.doJSON(ctx, http.MethodPost, "/secrets", map[string]any{"data": map[string]any{"value": "foo"}})
	if err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}

	logs := output.String()
	for _, want := range []string{
		`"@message":"tensordock api request"`,
		`"@message":"tensordock api request body"`,
		`"@message":"tensordock api response"`,
		`"@message":"tensordock api response body"`,
		`"method":"POST"`,
		`"url":"http://`,
		`/secrets"`,
		`"{\"data\":{\"value\":\"foo\"}}"`,
		`"{\"data\":{\"id\":\"secret-1\"}}"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected logs to contain %q, got: %s", want, logs)
		}
	}
}

func TestDoJSONEmitsTransportFailureLog(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:1", "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)

	_, err = client.doJSON(ctx, http.MethodGet, "/secrets", nil)
	if err == nil {
		t.Fatal("expected doJSON to fail")
	}

	logs := output.String()
	if !strings.Contains(logs, `"@message":"tensordock api transport failed"`) {
		t.Fatalf("expected transport failure log, got: %s", logs)
	}
}
