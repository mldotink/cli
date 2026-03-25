// Package ink provides a Go client for the Ink platform API (ml.ink).
package ink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	DefaultBaseURL = "https://api.ml.ink/graphql"
	DefaultExecURL = "wss://exec-eu-central-1.ml.ink"
)

// Config configures an Ink API client.
type Config struct {
	// APIKey is required. Create one at https://ml.ink/account/api-keys.
	APIKey string

	// BaseURL overrides the GraphQL endpoint. Default: https://api.ml.ink/graphql
	BaseURL string

	// ExecURL overrides the exec-proxy WebSocket endpoint.
	ExecURL string

	// HTTPClient overrides the HTTP client used for GraphQL requests.
	// Auth headers are added automatically.
	HTTPClient *http.Client
}

// Client is an Ink platform API client.
type Client struct {
	apiKey     string
	baseURL    string
	execURL    string
	httpClient *http.Client
}

// NewClient creates a new Ink API client.
func NewClient(cfg Config) *Client {
	if cfg.APIKey == "" {
		panic("ink: APIKey is required")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	execURL := cfg.ExecURL
	if execURL == "" {
		execURL = DefaultExecURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	httpClient.Transport = &authTransport{
		apiKey: cfg.APIKey,
		base:   transportOrDefault(httpClient.Transport),
	}
	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		execURL:    execURL,
		httpClient: httpClient,
	}
}

// BaseURL returns the configured GraphQL endpoint.
func (c *Client) BaseURL() string { return c.baseURL }

// ExecBaseURL returns the configured exec-proxy WebSocket base URL.
func (c *Client) ExecBaseURL() string { return c.execURL }

// APIKey returns the configured API key.
func (c *Client) APIKey() string { return c.apiKey }

type authTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.base.RoundTrip(req)
}

func transportOrDefault(t http.RoundTripper) http.RoundTripper {
	if t != nil {
		return t
	}
	return http.DefaultTransport
}

// Error is a GraphQL error returned by the Ink API.
type Error struct {
	Message    string         `json:"message"`
	Path       []string       `json:"path"`
	Extensions map[string]any `json:"extensions"`
}

func (e *Error) Error() string { return e.Message }

// Errors is a list of GraphQL errors.
type Errors []*Error

func (e Errors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Message
	}
	return strings.Join(msgs, "; ")
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors Errors          `json:"errors"`
}

func (c *Client) doGraphQL(ctx context.Context, query string, vars map[string]any, result any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: vars})
	if err != nil {
		return fmt.Errorf("ink: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ink: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ink: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ink: unexpected status %d", resp.StatusCode)
	}

	var gqlResp graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return fmt.Errorf("ink: decode response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return gqlResp.Errors
	}
	if result != nil {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("ink: decode data: %w", err)
		}
	}
	return nil
}
