package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseLinkID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in    string
		want  string
		bad   bool
	}{
		{in: "42", want: "42"},
		{in: "../collections/7", bad: true},
		{in: "1/../../users", bad: true},
		{in: "..%2f..%2fusers", bad: true},
		{in: "1?x=..", bad: true},
		{in: "0", bad: true},
		{in: "-1", bad: true},
		{in: "abc", bad: true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := parseLinkID(tc.in)
			if tc.bad {
				if err == nil {
					t.Fatalf("expected error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLinkID(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestLinksIDPathDryRun(t *testing.T) {
	getenv := func(k string) string {
		if k == "LINKWARDEN_URL" {
			return "https://lw.test"
		}
		return ""
	}

	t.Run("valid get", func(t *testing.T) {
		var out, errOut strings.Builder
		code := Execute([]string{"links", "get", "42", "--dry-run"}, strings.NewReader(""), &out, &errOut, getenv)
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		var v map[string]any
		if err := json.Unmarshal([]byte(out.String()), &v); err != nil {
			t.Fatal(err)
		}
		if v["url"] != "https://lw.test/api/v1/links/42" {
			t.Fatalf("url=%v", v["url"])
		}
	})

	bad := []struct {
		cmd  []string
		name string
	}{
		{[]string{"links", "get", "../collections/7", "--dry-run"}, "get traversal"},
		{[]string{"links", "delete", "1/../../users", "--dry-run", "--allow-write"}, "delete traversal"},
		{[]string{"links", "update", "..%2f..%2fusers", "--dry-run", "--allow-write", "--json", `{}`}, "update encoded"},
		{[]string{"links", "get", "1?x=..", "--dry-run"}, "get query injection"},
	}
	for _, tc := range bad {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut strings.Builder
			code := Execute(tc.cmd, strings.NewReader(""), &out, &errOut, getenv)
			if code != 2 {
				t.Fatalf("code=%d want 2 stderr=%s stdout=%s", code, errOut.String(), out.String())
			}
		})
	}
}
