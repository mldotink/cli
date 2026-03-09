package api

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

const DefaultEndpoint = "https://api.ml.ink/graphql"

type authTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.base.RoundTrip(req)
}

func NewClient(apiKey string) graphql.Client {
	httpClient := &http.Client{
		Transport: &authTransport{apiKey: apiKey, base: http.DefaultTransport},
	}
	return graphql.NewClient(DefaultEndpoint, httpClient)
}
