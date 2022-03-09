package openapistackql

import (
	"embed"
	"fmt"
	"io"
	"path"
)

//go:embed embeddedproviders/googleapis.com/*
var googleProvider embed.FS

//go:embed embeddedproviders/okta/*
var oktaProvider embed.FS

func GetEmbeddedProvider(prov string) (embed.FS, error) {
	return getEmbeddedProvider(prov)
}

func getEmbeddedProvider(prov string) (embed.FS, error) {
	switch prov {
	case "google":
		return googleProvider, nil
	case "okta":
		return oktaProvider, nil
	}
	return embed.FS{}, fmt.Errorf("no such embedded provider: '%s'", prov)
}

func GetEmbeddedDist(prov, version string) (io.ReadCloser, error) {
	return getEmbeddedDist(prov, version)
}

func getEmbeddedDist(prov, version string) (io.ReadCloser, error) {
	pr, err := getEmbeddedProvider(prov)
	if err != nil {
		return nil, err
	}
	if prov == "google" {
		prov = "googleapis.com"
	}
	return pr.Open(path.Join(prov, fmt.Sprintf("%s.tgz", version)))
}
