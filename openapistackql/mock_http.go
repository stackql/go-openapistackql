package openapistackql

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

var (
//
)

type SimpleMockRegistryRoundTripper struct {
	fileRoot     string
	registryRoot *url.URL
}

func NewSimpleMockRegistryRoundTripper(fileRoot string, registryRoot *url.URL) *SimpleMockRegistryRoundTripper {
	return &SimpleMockRegistryRoundTripper{
		fileRoot:     fileRoot,
		registryRoot: registryRoot,
	}
}

func (rt *SimpleMockRegistryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	fp, err := GetFilePathFromRepositoryRoot(path.Join(rt.fileRoot, strings.TrimPrefix(req.URL.Path, rt.registryRoot.Path)))
	if err != nil {
		return nil, err
	}
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       f,
	}
	return resp, nil
}

func getMockRoundTripper(registryUrl string) (http.RoundTripper, error) {
	u, err := url.Parse(registryUrl)
	if err != nil {
		return nil, err
	}
	return NewSimpleMockRegistryRoundTripper("test/registry/src", u), nil
}

func getMockHttpRegistry(useEmbedded bool) (RegistryAPI, error) {
	rt, err := getMockRoundTripper(defaultRegistryUrlString)
	if err != nil {
		return nil, err
	}
	return NewRegistry(defaultRegistryUrlString, rt, useEmbedded)
}

func getMockFileRegistry(registryRoot string, useEmbedded bool) (RegistryAPI, error) {
	return NewRegistry(registryRoot, nil, useEmbedded)
}

func getMockEmbeddedRegistry() (RegistryAPI, error) {
	return getMockHttpRegistry(true)
}

func getMockRemoteRegistry() (RegistryAPI, error) {
	return getMockHttpRegistry(false)
}

func getMockLocalRegistry() (RegistryAPI, error) {
	localRegPath, err := GetFilePathFromRepositoryRoot("test/registry/src")
	if err != nil {
		return nil, err
	}
	return getMockFileRegistry(fmt.Sprintf("file://%s", localRegPath), false)
}

func GetMockEmbeddedRegistry() (RegistryAPI, error) {
	return getMockEmbeddedRegistry()
}

func GetMockRegistry() (RegistryAPI, error) {
	return getMockRemoteRegistry()
}

func GetMockLocalRegistry() (RegistryAPI, error) {
	return getMockLocalRegistry()
}
