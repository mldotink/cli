package api

import (
	"net/http"

	ink "github.com/mldotink/sdk-go"
)

const clientHeader = "ink-cli"

type ClientConfig struct {
	APIKey  string
	BaseURL string
}

func NewClient(cfg ClientConfig) *ink.Client {
	return ink.NewClient(ink.Config{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		HTTPClient: &http.Client{
			Transport: clientHeaderTransport{base: http.DefaultTransport},
		},
	})
}

type clientHeaderTransport struct {
	base http.RoundTripper
}

func (t clientHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Ink-Client", clientHeader)
	return t.base.RoundTrip(req)
}
