package openapistackql

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	defaultRegistryUrlString string = "https://raw.githubusercontent.com/stackql/stackql-provider-registry/intial-devel/providers/src"
	httpSchemeRegexpString   string = `(?i)^https?$`
	fileSchemeRegexpString   string = `(?i)^file?$`
)

var (
	httpSchemeRegexp *regexp.Regexp = regexp.MustCompile(httpSchemeRegexpString)
	fileSchemeRegexp *regexp.Regexp = regexp.MustCompile(fileSchemeRegexpString)
)

type RegistryAPI interface {
	GetDocBytes(string) ([]byte, error)
	GetResourcesShallowFromProvider(*Provider, string) (*ResourceRegister, error)
	GetResourcesShallowFromProviderService(*ProviderService) (*ResourceRegister, error)
	GetResourcesShallowFromURL(string) (*ResourceRegister, error)
	GetService(string) (*Service, error)
	GetServiceFragment(*ProviderService, string) (*Service, error)
	GetServiceFromProviderService(*ProviderService) (*Service, error)
	GetServiceDocBytes(string) ([]byte, error)
	GetResourcesRegisterDocBytes(string) ([]byte, error)
	LoadProviderByName(string, string) (*Provider, error)
}

type Registry struct {
	regUrl      *url.URL
	transport   http.RoundTripper
	useEmbedded bool
}

func NewRegistry(registryUrl string, transport http.RoundTripper, useEmbedded bool) (RegistryAPI, error) {
	return newRegistry(registryUrl, transport, useEmbedded)
}

func newRegistry(registryUrl string, transport http.RoundTripper, useEmbedded bool) (RegistryAPI, error) {
	if registryUrl == "" {
		registryUrl = defaultRegistryUrlString
	}
	regUrl, err := url.Parse(registryUrl)
	if err != nil {
		return nil, err
	}
	return &Registry{
		regUrl:      regUrl,
		transport:   transport,
		useEmbedded: useEmbedded,
	}, nil
}

func (r *Registry) isHttp() bool {
	return httpSchemeRegexp.MatchString(r.regUrl.Scheme)
}

func (r *Registry) isFile() bool {
	return fileSchemeRegexp.MatchString(r.regUrl.Scheme)
}

func (r *Registry) isLocalFile() bool {
	return r.isFile() && strings.HasPrefix(r.regUrl.Path, "/")
}

func (r *Registry) GetDocBytes(docPath string) ([]byte, error) {
	return r.getDocBytes(docPath)
}

func (r *Registry) getProviderDocBytes(prov string, version string) ([]byte, error) {
	switch prov {
	case "google":
		prov = "googleapis.com"
	}
	return r.getDocBytes(path.Join(prov, version, "provider.yaml"))
}

func (r *Registry) LoadProviderByName(prov string, version string) (*Provider, error) {
	if r.useEmbedded {
		return LoadProviderByName(prov)
	}
	b, err := r.getProviderDocBytes(prov, version)
	if err != nil {
		return nil, err
	}
	return LoadProviderDocFromBytes(b)
}

func (r *Registry) GetServiceDocBytes(url string) ([]byte, error) {
	return r.getDocBytes(url)
}

func (r *Registry) GetResourcesRegisterDocBytes(url string) ([]byte, error) {
	return r.getDocBytes(url)
}

func (r *Registry) GetService(url string) (*Service, error) {
	b, err := r.getDocBytes(url)
	if err != nil {
		return nil, err
	}
	return LoadServiceDocFromBytes(b)
}

func (r *Registry) GetResourcesShallowFromProvider(pr *Provider, serviceKey string) (*ResourceRegister, error) {
	if r.useEmbedded {
		return pr.GetResourcesShallow(serviceKey)
	}
	return pr.getResourcesShallowWithRegistry(r, serviceKey)
}

func (r *Registry) GetResourcesShallowFromProviderService(pr *ProviderService) (*ResourceRegister, error) {
	if r.useEmbedded {
		return pr.GetResourcesShallow()
	}
	return pr.getResourcesShallowWithRegistry(r)
}

func (r *Registry) GetResourcesShallowFromURL(url string) (*ResourceRegister, error) {
	b, err := r.getDocBytes(url)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return loadResourcesShallow(b)
}

func (r *Registry) GetServiceFromProviderService(ps *ProviderService) (*Service, error) {
	if ps.ServiceRef == nil || ps.ServiceRef.Ref == "" {
		return nil, fmt.Errorf("no service reachable for %s", ps.GetName())
	}
	return r.GetService(ps.ServiceRef.Ref)
}

func (r *Registry) GetServiceFragment(ps *ProviderService, resourceKey string) (*Service, error) {

	if ps.ResourcesRef == nil || ps.ResourcesRef.Ref == "" {
		if ps.ServiceRef == nil || ps.ServiceRef.Ref == "" {
			return nil, fmt.Errorf("no service or resources reachable for %s", ps.GetName())
		}
		return r.GetService(ps.ServiceRef.Ref)
	}
	rr, err := r.GetResourcesShallowFromProviderService(ps)
	if err != nil {
		return nil, err
	}
	rsc, ok := rr.Resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("cannot locate resource for key = '%s'", resourceKey)
	}
	sdRef := ps.getServiceDocRef(rr, rsc)
	if sdRef.Ref == "" {
		return nil, fmt.Errorf("no service doc available for resourceKey = '%s'", resourceKey)
	}
	if sdRef.Value != nil {
		return sdRef.Value, nil
	}
	sb, err := r.getDocBytes(sdRef.Ref)
	if err != nil {
		return nil, err
	}
	svc, err := LoadServiceSubsetDocFromBytes(rr, resourceKey, sb)
	if err != nil {
		return nil, err
	}
	ps.ServiceRef.Value = svc
	return ps.ServiceRef.Value, nil
}

func (r *Registry) getDocBytes(docPath string) ([]byte, error) {
	if r.useEmbedded {
		return getServiceDocBytes(docPath)
	}
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
	if r.isLocalFile() {
		b, err := os.ReadFile(path.Join(r.regUrl.Path, docPath))
		if err != nil {
			return nil, fmt.Errorf("cannot read local registry file: '%s'", err.Error())
		}
		return b, nil
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}
