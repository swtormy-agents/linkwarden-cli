package linkwarden

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDoSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("bad auth: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/links" {
			t.Fatalf("path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":[]}`))
	}))
	defer srv.Close()

	c := &Client{Token: "secret-token", BaseURL: srv.URL, HTTP: srv.Client()}
	raw, err := c.Do(context.Background(), Request{Method: http.MethodGet, Path: "/api/v1/links"})
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]any
	if err := json.Unmarshal(raw, &page); err != nil || page["response"] == nil {
		t.Fatalf("got %s", raw)
	}
}

func TestClientRedact(t *testing.T) {
	c := &Client{Token: "abc123"}
	if got := c.Redact("Bearer abc123 leaked"); got != "Bearer <TOKEN> leaked" {
		t.Fatalf("redact: %q", got)
	}
}

func TestDryRunMasksToken(t *testing.T) {
	d := DryRun("https://lw.test", Request{Method: "GET", Path: "/api/v1/tags"})
	h := d["header"].(map[string]string)
	if h["Authorization"] != "Bearer <TOKEN>" {
		t.Fatalf("header %v", h)
	}
}

func TestClientDoAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer srv.Close()

	c := &Client{Token: "bad", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Do(context.Background(), Request{Method: http.MethodGet, Path: "/api/v1/tags"})
	var api *APIError
	if !errors.As(err, &api) || !api.IsAuth() {
		t.Fatalf("expected auth error, got %v", err)
	}
}
