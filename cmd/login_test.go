package cmd

import "testing"

func TestParseOAuthCallbackInputURL(t *testing.T) {
	res, err := parseOAuthCallbackInput("http://127.0.0.1:1234/callback?code=abc123&state=state123", "state123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.code != "abc123" {
		t.Fatalf("expected code abc123, got %q", res.code)
	}
}

func TestParseOAuthCallbackInputRawCode(t *testing.T) {
	res, err := parseOAuthCallbackInput("eyJhbGciOi...", "ignored")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.code != "eyJhbGciOi..." {
		t.Fatalf("expected raw code to pass through, got %q", res.code)
	}
}

func TestParseOAuthCallbackInputRejectsWrongState(t *testing.T) {
	_, err := parseOAuthCallbackInput("http://127.0.0.1:1234/callback?code=abc123&state=wrong", "expected")
	if err == nil {
		t.Fatal("expected state mismatch error")
	}
}

func TestParseOAuthCallbackInputRejectsMissingCode(t *testing.T) {
	_, err := parseOAuthCallbackInput("http://127.0.0.1:1234/callback?state=state123", "state123")
	if err == nil {
		t.Fatal("expected missing code error")
	}
}
