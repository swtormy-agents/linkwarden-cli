package linkwarden

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ValidateBaseURL checks scheme (http/https) and requires a host.
// Returns the URL without a trailing slash. insecureHTTP is true when scheme is http.
func ValidateBaseURL(raw string) (normalized string, insecureHTTP bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, errors.New("LINKWARDEN_URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, fmt.Errorf("LINKWARDEN_URL is not a valid URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https":
	default:
		return "", false, fmt.Errorf("LINKWARDEN_URL must use http or https (got %q)", u.Scheme)
	}
	if u.Host == "" {
		return "", false, errors.New("LINKWARDEN_URL must include a host")
	}
	if u.User != nil {
		return "", false, errors.New("LINKWARDEN_URL must not include userinfo")
	}
	normalized = strings.TrimRight(raw, "/")
	return normalized, scheme == "http", nil
}
