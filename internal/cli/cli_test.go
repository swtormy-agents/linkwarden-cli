package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testEnv(key string) string {
	switch key {
	case "LINKWARDEN_TOKEN":
		return "test-token-secret"
	case "LINKWARDEN_URL":
		return ""
	case "LINKWARDEN_ALLOW_WRITE":
		return "true"
	default:
		return ""
	}
}

func runCLI(t *testing.T, srvURL string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	getenv := testEnv
	if srvURL != "" {
		prev := getenv
		getenv = func(k string) string {
			if k == "LINKWARDEN_URL" {
				return srvURL
			}
			return prev(k)
		}
	}
	var out, errOut bytes.Buffer
	code = Execute(args, strings.NewReader(""), &out, &errOut, getenv)
	return out.String(), errOut.String(), code
}

func TestLinksList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token-secret" {
			t.Fatalf("bad auth")
		}
		if r.URL.Path != "/api/v1/links" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":[{"id":1,"url":"https://a"}]}`))
	}))
	defer srv.Close()

	out, errOut, code := runCLI(t, srv.URL, "links", "list")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	var page map[string]any
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("stdout: %s", out)
	}
}

func TestCreateBlockedWithoutWrite(t *testing.T) {
	getenv := func(k string) string {
		if k == "LINKWARDEN_URL" {
			return "https://example.test"
		}
		if k == "LINKWARDEN_TOKEN" {
			return "tok"
		}
		return ""
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"links", "create", "--json", `{"url":"https://x"}`}, strings.NewReader(""), &out, &errOut, getenv)
	if code != 4 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), `"type":"policy"`) {
		t.Fatalf("stderr %s", errOut.String())
	}
}

func TestDryRun(t *testing.T) {
	getenv := func(k string) string {
		if k == "LINKWARDEN_URL" {
			return "https://lw.test"
		}
		return ""
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"links", "list", "--dry-run"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out.String()), &v); err != nil {
		t.Fatal(err)
	}
	h := v["header"].(map[string]any)
	if h["Authorization"] != "Bearer <TOKEN>" {
		t.Fatalf("%v", h)
	}
}

func TestReportErrorRedactsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"message":"upstream failed"}`))
	}))
	defer srv.Close()
	getenv := func(k string) string {
		switch k {
		case "LINKWARDEN_URL":
			return srv.URL
		case "LINKWARDEN_TOKEN":
			return "super-secret-token-value"
		default:
			return ""
		}
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"links", "list"}, strings.NewReader(""), &out, &errOut, getenv)
	if code == 0 {
		t.Fatal("expected failure")
	}
	combined := out.String() + errOut.String()
	if strings.Contains(combined, "super-secret-token-value") {
		t.Fatalf("token leaked in output: %s", combined)
	}
}

func TestHTTPURLWarning(t *testing.T) {
	getenv := func(k string) string {
		if k == "LINKWARDEN_URL" {
			return "http://127.0.0.1:3000"
		}
		return ""
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"links", "list", "--dry-run"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "http://") {
		t.Fatalf("expected http warning, stderr=%q", errOut.String())
	}
}

func TestInvalidURLRejected(t *testing.T) {
	getenv := func(k string) string {
		if k == "LINKWARDEN_URL" {
			return "not-a-url"
		}
		return ""
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"links", "list", "--dry-run"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != 2 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
}

func TestVersionJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Execute([]string{"version"}, strings.NewReader(""), &out, &errOut, testEnv)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out.String()), &v); err != nil {
		t.Fatalf("stdout %s", out.String())
	}
	if v["version"] == nil {
		t.Fatalf("missing version: %v", v)
	}
}
