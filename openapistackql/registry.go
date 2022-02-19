package openapistackql

import (
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/stackql/stackql-provider-registry/signing/Ed25519/app/edcrypto"
)

const (
	defaultRegistryUrlString string = "https://raw.githubusercontent.com/stackql/stackql-provider-registry/intial-devel/providers"
	defaultSrcPrefix         string = "src"
	defaultDistPrefix        string = "dist"
	httpSchemeRegexpString   string = `(?i)^https?$`
	fileSchemeRegexpString   string = `(?i)^file?$`
)

var (
	httpSchemeRegexp *regexp.Regexp = regexp.MustCompile(httpSchemeRegexpString)
	fileSchemeRegexp *regexp.Regexp = regexp.MustCompile(fileSchemeRegexpString)
)

type RegistryAPI interface {
	PullArchive(archivePath string) (io.ReadCloser, error)
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

type RegistryConfig struct {
	RegistryURL   string                   `json:"url" yaml:"url"`
	SrcPrefix     *string                  `json:"srcPrefix" yaml:"srcPrefix"`
	DistPrefix    *string                  `json:"distPrefix" yaml:"distPrefix"`
	UseEmbedded   *bool                    `json:"useEmbedded" yaml:"useEmbedded"`
	LocalDocRoot  string                   `json:"localDocRoot" yaml:"localDocRoot"`
	VerfifyConfig *edcrypto.VerifierConfig `json:"verifyConfig" yaml:"verifyConfig"`
}

type Registry struct {
	regUrl       *url.URL
	srcUrl       *url.URL
	distUrl      *url.URL
	srcPrefix    string
	localDocRoot string
	transport    http.RoundTripper
	useEmbedded  bool
	verifier     *edcrypto.Verifier
}

func NewRegistry(registryCfg RegistryConfig, transport http.RoundTripper) (RegistryAPI, error) {
	return newRegistry(registryCfg, transport)
}

func newRegistry(registryCfg RegistryConfig, transport http.RoundTripper) (RegistryAPI, error) {
	registryUrl := registryCfg.RegistryURL
	if registryUrl == "" {
		registryUrl = defaultRegistryUrlString
	}
	useEmbedded := true // default
	if registryCfg.UseEmbedded != nil {
		useEmbedded = *registryCfg.UseEmbedded
	}
	srcUrlStr := registryUrl
	srcPrefix := ""
	if registryCfg.SrcPrefix == nil {
		srcPrefix = defaultSrcPrefix
	} else {
		srcPrefix = *registryCfg.SrcPrefix
	}
	if srcPrefix != "" {
		srcUrlStr = fmt.Sprintf("%s/%s", registryUrl, srcPrefix)
	}
	distUrlStr := registryUrl
	distPrefix := ""
	if registryCfg.DistPrefix == nil {
		distPrefix = defaultDistPrefix
	} else {
		distPrefix = *registryCfg.DistPrefix
	}
	if distPrefix != "" {
		distUrlStr = fmt.Sprintf("%s/%s", registryUrl, distPrefix)
	}
	regUrl, err := url.Parse(registryUrl)
	if err != nil {
		return nil, err
	}
	srcUrl, err := url.Parse(srcUrlStr)
	if err != nil {
		return nil, err
	}
	distUrl, err := url.Parse(distUrlStr)
	if err != nil {
		return nil, err
	}
	var ver *edcrypto.Verifier
	if registryCfg.VerfifyConfig == nil {
		ver, err = edcrypto.NewVerifier(edcrypto.NewVerifierConfig("", "", ""))
	} else {
		ver, err = edcrypto.NewVerifier(*registryCfg.VerfifyConfig)
	}
	if err != nil {
		return nil, err
	}
	return &Registry{
		regUrl:       regUrl,
		srcUrl:       srcUrl,
		distUrl:      distUrl,
		localDocRoot: registryCfg.LocalDocRoot,
		transport:    transport,
		useEmbedded:  useEmbedded,
		verifier:     ver,
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
	return r.getVerifiedDocBytes(docPath)
}

func (r *Registry) getProviderDocBytes(prov string, version string) ([]byte, error) {
	switch prov {
	case "google":
		prov = "googleapis.com"
	}
	return r.getVerifiedDocBytes(path.Join(prov, version, "provider.yaml"))
}

func (r *Registry) PullProviderArchive(prov string, version string) (io.ReadCloser, error) {
	switch prov {
	case "google":
		prov = "googleapis.com"
	}
	fp := path.Join("dist", prov, fmt.Sprintf("%s.tgz", version))
	return r.pullArchive(fp)
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
	return r.getVerifiedDocBytes(url)
}

func (r *Registry) GetResourcesRegisterDocBytes(url string) ([]byte, error) {
	return r.getVerifiedDocBytes(url)
}

func (r *Registry) GetService(url string) (*Service, error) {
	b, err := r.getVerifiedDocBytes(url)
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
	b, err := r.getVerifiedDocBytes(url)
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
	sb, err := r.getVerifiedDocBytes(sdRef.Ref)
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

func (r *Registry) checkSignature(docUrl string, verFile, sigFile io.ReadCloser) (*edcrypto.VerifierResponse, error) {
	if sigFile == nil {
		return nil, fmt.Errorf("nil signature")
	}
	vc := edcrypto.NewVerifyContext(docUrl, sigFile, verFile, "base64", true, x509.VerifyOptions{})
	vr, err := r.verifier.VerifyFileFromCertificateBytes(vc)
	return &vr, err
}

func (r *Registry) PullArchive(archivePath string) (io.ReadCloser, error) {
	return r.pullArchive(archivePath)
}

func (r *Registry) pullArchive(archivePath string) (io.ReadCloser, error) {
	return r.getUnVerifiedDoc(archivePath)
}

func (r *Registry) getRemoteDoc(docPath string) (io.ReadCloser, error) {
	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(path.Join(r.srcUrl.Path, docPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *Registry) getLocalDoc(docPath string) (io.ReadCloser, error) {
	localPath := path.Join(r.localDocRoot, docPath)
	fi, err := os.Open(localPath)
	if err != nil {
		if fi != nil {
			fi.Close()
		}
		return nil, err
	}
	if fi == nil {
		return nil, fmt.Errorf("nil file")
	}
	return fi, nil
}

func (r *Registry) getUnVerifiedDoc(docPath string) (io.ReadCloser, error) {
	if r.useEmbedded {
		return getServiceDoc(docPath)
	}
	if r.isLocalFile() {
		return os.Open(path.Join(r.srcUrl.Path, docPath))
	}
	if r.localDocRoot != "" {
		localPath := path.Join(r.localDocRoot, docPath)
		lf, err := r.getLocalDoc(localPath)
		if err == nil {
			return lf, nil
		}
	}
	if r.isHttp() {
		cl := &http.Client{}
		if r.transport != nil {
			cl.Transport = r.transport
		}
		return r.getRemoteDoc(docPath)
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}

func (r *Registry) getVerifiedDocResponse(docPath string) (*edcrypto.VerifierResponse, error) {
	if r.useEmbedded {
		lf, err := getServiceDoc(docPath)
		if err != nil {
			return nil, err
		}
		sf, err := r.getLocalDoc(fmt.Sprintf("%s.sig", docPath))
		if err != nil {
			lf.Close()
			return nil, fmt.Errorf("embedded document present but signature file not present")
		}
		return r.checkSignature(docPath, lf, sf)
	}
	if r.isLocalFile() {
		rb, err := os.Open(path.Join(r.srcUrl.Path, docPath))
		if err != nil {
			return nil, fmt.Errorf("cannot read local registry file: '%s'", err.Error())
		}
		sb, err := os.Open(path.Join(r.srcUrl.Path, fmt.Sprintf("%s.sig", docPath)))
		if err != nil {
			return nil, fmt.Errorf("cannot read local signature file: '%s'", err.Error())
		}
		return r.checkSignature(docPath, rb, sb)
	}
	if r.localDocRoot != "" {
		localPath := path.Join(r.localDocRoot, docPath)
		lf, err := r.getLocalDoc(localPath)
		if err != nil {
			sf, err := r.getLocalDoc(fmt.Sprintf("%s.sig", localPath))
			if err != nil {
				lf.Close()
				return nil, fmt.Errorf("local document present but signature file not present")
			}
			return r.checkSignature(localPath, lf, sf)
		}
	}
	fullUrl, err := url.Parse(r.regUrl.String())
	if err != nil {
		return nil, err
	}
	fullUrl.Path = path.Join(fullUrl.Path, docPath)
	verifyUrl := fullUrl.String()
	if r.isHttp() {
		cl := &http.Client{}
		if r.transport != nil {
			cl.Transport = r.transport
		}
		response, err := r.getRemoteDoc(docPath)
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, fmt.Errorf("no response body from remote")
		}
		sigResponse, err := r.getRemoteDoc(fmt.Sprintf("%s.sig", docPath))
		if err != nil {
			response.Close()
			return nil, fmt.Errorf("remote document '%s' present but signature file not present", verifyUrl)
		}
		return r.checkSignature(verifyUrl, response, sigResponse)
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}

func (r *Registry) getVerifiedDocBytes(docPath string) ([]byte, error) {
	if r.useEmbedded {
		return getServiceDocBytes(docPath)
	}
	vr, err := r.getVerifiedDocResponse(docPath)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(vr.VerifyFile)
}
