package api

import ink "github.com/mldotink/sdk-go"

func NewClient(apiKey string) *ink.Client {
	return ink.NewClient(ink.Config{APIKey: apiKey})
}
