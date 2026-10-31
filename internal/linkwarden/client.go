// Package linkwarden is a minimal HTTP client for the Linkwarden REST API.
package linkwarden

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	Token   string
	BaseURL string
	HTTP    *http.Client
}

type APIError struct {
	Status  int
	Body    json.RawMessage
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("linkwarden API HTTP %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("linkwarden API HTTP %d", e.Status)
}

func (e *APIError) IsAuth() bool {
	return e.Status == 401 || e.Status == 403
}

type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Body        []byte
	ContentType string
}

func (c *Client) baseURL() string {
	return strings.TrimRight(c.BaseURL, "/")
}

func (c *Client) http() *http.Client {
	if c.HTTP == nil {
		return http.DefaultClient
	}
	return c.HTTP
}

func (c *Client) Redact(s string) string {
	if c.Token == "" {
		return s
	}
	return strings.ReplaceAll(s, c.Token, "<TOKEN>")
}

func authHeader(token string) string {
	return "Bearer " + token
}

// DryRun describes the request that Do would send, without credentials.
func DryRun(baseURL string, r Request) map[string]any {
	out := map[string]any{
		"method": r.Method,
		"url":    strings.TrimRight(baseURL, "/") + r.Path,
		"header": map[string]string{"Authorization": "Bearer <TOKEN>"},
	}
	if len(r.Query) > 0 {
		out["query"] = r.Query.Encode()
	}
	if len(r.Body) > 0 {
		var body any
		if json.Unmarshal(r.Body, &body) == nil {
			out["body"] = body
		} else {
			out["body"] = string(r.Body)
		}
	}
	return out
}

func (c *Client) Do(ctx context.Context, r Request) (json.RawMessage, error) {
	if c.Token == "" {
		return nil, errors.New("LINKWARDEN_TOKEN is not set (create a session token or set config.json with mode 0600)")
	}
	if c.baseURL() == "" {
		return nil, errors.New("LINKWARDEN_URL is not set")
	}
	u, err := url.Parse(c.baseURL() + r.Path)
	if err != nil {
		return nil, err
	}
	if len(r.Query) > 0 {
		u.RawQuery = r.Query.Encode()
	}
	var body io.Reader
	ct := r.ContentType
	if len(r.Body) > 0 {
		body = bytes.NewReader(r.Body)
		if ct == "" {
			ct = "application/json"
		}
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), body)
	if err != nil {
		return nil, errors.New(c.Redact(err.Error()))
	}
	req.Header.Set("Authorization", authHeader(c.Token))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", c.Redact(err.Error()))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %s", c.Redact(err.Error()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := c.Redact(extractMessage(data))
		return nil, &APIError{Status: resp.StatusCode, Body: data, Message: msg}
	}
	if len(data) == 0 {
		return json.RawMessage(`null`), nil
	}
	if !json.Valid(data) {
		return nil, &APIError{Status: resp.StatusCode, Message: "non-JSON response"}
	}
	return json.RawMessage(data), nil
}

func extractMessage(data []byte) string {
	var v struct {
		Message  any `json:"message"`
		Response any `json:"response"`
	}
	if json.Unmarshal(data, &v) == nil {
		if v.Message != nil {
			b, _ := json.Marshal(v.Message)
			return string(b)
		}
		if s, ok := v.Response.(string); ok && s != "" {
			return s
		}
	}
	if len(data) > 512 {
		return string(data[:512]) + "..."
	}
	return string(data)
}
