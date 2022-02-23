package openapistackql

import (
	"embed"
	"fmt"
)

//go:embed embeddedproviders/googleapis.com/v1/* embeddedproviders/googleapis.com/v1/services/* embeddedproviders/googleapis.com/v1/services-split/*/* embeddedproviders/googleapis.com/v1/resources/*
var googleProvider embed.FS

//go:embed embeddedproviders/okta/v1/* embeddedproviders/okta/v1/*/*
var oktaProvider embed.FS

func GetEmbeddedProvider(prov string) (embed.FS, error) {
	switch prov {
	case "google":
		return googleProvider, nil
	case "okta":
		return oktaProvider, nil
	}
	return embed.FS{}, fmt.Errorf("no such embedded provider: '%s'", prov)
}

func ListEmbeddedProviders() []string {
	return []string{
		"google",
		"okta",
	}
}
