package openapistackql

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
)

const (
	defaultRegistryUrlString string = "https://raw.githubusercontent.com/stackql/stackql-provider-registry/intial-devel/providers/src"
	httpSchemeRegexpString   string = `(?i)^https?$`
)

var (
	httpSchemeRegexp *regexp.Regexp = regexp.MustCompile(httpSchemeRegexpString)
)

type Registry struct {
	regUrl    *url.URL
	transport *http.Transport
}

func NewRegistry(registryUrl string, transport *http.Transport) (*Registry, error) {
	return newRegistry(registryUrl, transport)
}

func newRegistry(registryUrl string, transport *http.Transport) (*Registry, error) {
	if registryUrl == "" {
		registryUrl = defaultRegistryUrlString
	}
	regUrl, err := url.Parse(registryUrl)
	if err != nil {
		return nil, err
	}
	return &Registry{
		regUrl:    regUrl,
		transport: transport,
	}, nil
}

func (r *Registry) isHttp() bool {
	return httpSchemeRegexp.MatchString(r.regUrl.Scheme)
}

func (r *Registry) GetDocBytes(docPath string) ([]byte, error) {
	return r.getDocBytes(docPath)
}

func (r *Registry) GetProviderDocBytes(prov string, version string) ([]byte, error) {
	switch prov {
	case "google":
		prov = "googleapis.com"
	}
	return r.getDocBytes(path.Join(prov, version, "provider.yaml"))
}

func (r *Registry) GetServiceDocBytes(url string) ([]byte, error) {
	return r.getDocBytes(url)
}

func (r *Registry) GetResourcesRegisterDocBytes(url string) ([]byte, error) {
	return r.getDocBytes(url)
}

func (r *Registry) getDocBytes(docPath string) ([]byte, error) {
	if r.isHttp() {
		cl := &http.Client{}
		if r.transport != nil {
			cl.Transport = r.transport
		}
		response, err := cl.Get(path.Join(r.regUrl.Path, docPath))
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		return io.ReadAll(response.Body)
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}
