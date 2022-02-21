package openapistackql

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/stackql/go-openapistackql/pkg/fileutil"
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
	fp, err := fileutil.GetForwardSlashFilePathFromRepositoryRoot(path.Join(rt.fileRoot, strings.TrimPrefix(req.URL.Path, rt.registryRoot.Path)))
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
	return NewSimpleMockRegistryRoundTripper("test/registry", u), nil
}

func getMockHttpRegistry(vc RegistryConfig) (RegistryAPI, error) {
	rt, err := getMockRoundTripper(defaultRegistryUrlString)
	if err != nil {
		return nil, err
	}
	localRegPath, err := fileutil.GetForwardSlashFilePathFromRepositoryRoot("test/registry")
	if err != nil {
		return nil, err
	}
	return NewRegistry(RegistryConfig{RegistryURL: defaultRegistryUrlString, UseEmbedded: vc.UseEmbedded, LocalDocRoot: localRegPath, SrcPrefix: vc.SrcPrefix}, rt)
}

func getMockFileRegistry(vc RegistryConfig, registryRoot string, useEmbedded bool) (RegistryAPI, error) {
	localRegPath, err := fileutil.GetForwardSlashFilePathFromRepositoryRoot("test/registry")
	if err != nil {
		return nil, err
	}
	return NewRegistry(RegistryConfig{RegistryURL: registryRoot, UseEmbedded: &useEmbedded, LocalDocRoot: localRegPath, SrcPrefix: vc.SrcPrefix}, nil)
}

func getMockRemoteRegistry(vc RegistryConfig) (RegistryAPI, error) {
	return getMockHttpRegistry(vc)
}

func getMockLocalRegistry(vc RegistryConfig) (RegistryAPI, error) {
	localRegPath, err := fileutil.GetForwardSlashFilePathFromRepositoryRoot("test/registry")
	if err != nil {
		return nil, err
	}
	return getMockFileRegistry(vc, fmt.Sprintf("file://%s", localRegPath), false)
}

func GetMockRegistry(vc RegistryConfig) (RegistryAPI, error) {
	return getMockRemoteRegistry(vc)
}

func GetMockLocalRegistry(vc RegistryConfig) (RegistryAPI, error) {
	return getMockLocalRegistry(vc)
}
