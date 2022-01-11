package openapistackql

import (
	"embed"
	"fmt"
)

//go:embed embeddedproviders/googleapis.com/* embeddedproviders/googleapis.com/services/*
var googleProvider embed.FS

//go:embed embeddedproviders/okta/* embeddedproviders/okta/*/*
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
