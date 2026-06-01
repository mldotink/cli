package api

import (
	"net/http"
	"testing"
)

type captureTransport struct {
	header http.Header
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.header = req.Header.Clone()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestClientHeaderTransportAddsInkClient(t *testing.T) {
	capture := &captureTransport{}
	transport := clientHeaderTransport{base: capture}

	req, err := http.NewRequest(http.MethodPost, "https://api.ml.ink/graphql", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("round trip: %v", err)
	}

	if got := capture.header.Get("X-Ink-Client"); got != "ink-cli" {
		t.Fatalf("X-Ink-Client = %q, want ink-cli", got)
	}
}
