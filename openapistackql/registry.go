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

	"github.com/stackql/go-openapistackql/pkg/compression"
	"github.com/stackql/stackql-provider-registry/signing/Ed25519/app/edcrypto"
	"gopkg.in/yaml.v2"
)

const (
	defaultRegistryUrlString string = "https://raw.githubusercontent.com/stackql/stackql-provider-registry/intial-devel/providers"
	defaultSrcPrefix         string = "src"
	defaultDistPrefix        string = "dist"
	httpSchemeRegexpString   string = `(?i)^https?$`
	fileSchemeRegexpString   string = `(?i)^file$`
	remoteProviderListPath   string = `providers.yaml`
)

var (
	httpSchemeRegexp *regexp.Regexp = regexp.MustCompile(httpSchemeRegexpString)
	fileSchemeRegexp *regexp.Regexp = regexp.MustCompile(fileSchemeRegexpString)
)

type RegistryAPI interface {
	PullAndPersistProviderArchive(string, string) error
	PullProviderArchive(string, string) (io.ReadCloser, error)
	ListAllAvailableProviders() (map[string]ProviderDescription, error)
	ListLocallyAvailableProviders() map[string]struct{}
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
	RegistryURL      string                   `json:"url" yaml:"url"`
	SrcPrefix        *string                  `json:"srcPrefix" yaml:"srcPrefix"`
	DistPrefix       *string                  `json:"distPrefix" yaml:"distPrefix"`
	AllowSrcDownload bool                     `json:"allowSrcDownload" yaml:"allowSrcDownload"`
	LocalDocRoot     string                   `json:"localDocRoot" yaml:"localDocRoot"`
	VerfifyConfig    *edcrypto.VerifierConfig `json:"verifyConfig" yaml:"verifyConfig"`
}

type Registry struct {
	allowSrcDownload bool
	regUrl           *url.URL
	srcUrl           *url.URL
	distUrl          *url.URL
	localDocRoot     string
	localSrcPrefix   string
	localDistPrefix  string
	transport        http.RoundTripper
	useEmbedded      bool
	verifier         *edcrypto.Verifier
	nopVerifier      bool
}

func NewRegistry(registryCfg RegistryConfig, transport http.RoundTripper) (RegistryAPI, error) {
	return newRegistry(registryCfg, transport)
}

func newRegistry(registryCfg RegistryConfig, transport http.RoundTripper) (RegistryAPI, error) {
	registryUrl := registryCfg.RegistryURL
	if registryUrl == "" {
		registryUrl = defaultRegistryUrlString
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
	nopVerify := false
	if registryCfg.VerfifyConfig == nil {
		ver, err = edcrypto.NewVerifier(edcrypto.NewVerifierConfig("", "", ""))
	} else {
		ver, err = edcrypto.NewVerifier(*registryCfg.VerfifyConfig)
		nopVerify = registryCfg.VerfifyConfig.NopVerify
	}
	if err != nil {
		return nil, err
	}
	rv := &Registry{
		allowSrcDownload: registryCfg.AllowSrcDownload,
		regUrl:           regUrl,
		srcUrl:           srcUrl,
		distUrl:          distUrl,
		localDocRoot:     registryCfg.LocalDocRoot,
		localSrcPrefix:   srcPrefix,
		localDistPrefix:  distPrefix,
		transport:        transport,
		verifier:         ver,
		nopVerifier:      nopVerify,
	}
	return rv, nil
}

func (r *Registry) ListLocallyAvailableProviders() map[string]struct{} {
	rv := make(map[string]struct{})
	for k := range r.listLocalProviders() {
		rv[k] = struct{}{}
	}
	return rv
}

type ProviderDescription struct {
	Versions []string `json:"versions" yaml: "versions"`
}

type ProvidersList struct {
	Providers map[string]ProviderDescription `json:"providers" yaml: "providers"`
}

func NewProvidersList() ProvidersList {
	return ProvidersList{
		Providers: make(map[string]ProviderDescription),
	}
}

func (r *Registry) ListAllAvailableProviders() (map[string]ProviderDescription, error) {
	if r.isFile() {
		return nil, fmt.Errorf("'registry list' is meaningless in local mode")
	}
	rv := NewProvidersList()
	rc, err := r.getRemoteProviderList()
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, rv)
	if err != nil {
		return nil, err
	}
	return rv.Providers, nil
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
	return r.pullProviderArchive(prov, version)
}

func (r *Registry) pullProviderArchive(prov string, version string) (io.ReadCloser, error) {
	switch prov {
	case "google":
		prov = "googleapis.com"
	}
	fp := path.Join(prov, fmt.Sprintf("%s.tgz", version))
	return r.pullArchive(fp)
}

func (r *Registry) PullAndPersistProviderArchive(prov string, version string) error {
	return r.pullAndPersistProviderArchive(prov, version)
}

func (r *Registry) pullAndPersistProviderArchive(prov string, version string) error {
	if r.localDocRoot == "" {
		return fmt.Errorf("cannot pull provider without local doc location")
	}
	rdr, err := r.pullProviderArchive(prov, version)
	if err != nil {
		return err
	}
	pr := prov
	if pr == "google" {
		pr = "googleapis.com"
	}
	err = os.RemoveAll(path.Join(r.getLocalDocRoot(), pr, version))
	if err != nil {
		return err
	}
	return compression.DecompressToPath(rdr, path.Join(r.getLocalDocRoot(), pr))
}

func (r *Registry) LoadProviderByName(prov string, version string) (*Provider, error) {
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
	return pr.getResourcesShallowWithRegistry(r, serviceKey)
}

func (r *Registry) GetResourcesShallowFromProviderService(pr *ProviderService) (*ResourceRegister, error) {
	return pr.getResourcesShallowWithRegistry(r)
}

func (r *Registry) GetResourcesShallowFromURL(url string) (*ResourceRegister, error) {
	b, err := r.getVerifiedDocBytes(url)
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

func (r *Registry) pullArchive(archivePath string) (io.ReadCloser, error) {
	return r.getUnVerifiedArchive(archivePath)
}

func (r *Registry) getRemoteDoc(docPath string) (io.ReadCloser, error) {
	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(fmt.Sprintf("%s/%s", r.srcUrl.String(), docPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *Registry) getRemoteArchive(docPath string) (io.ReadCloser, error) {
	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(fmt.Sprintf("%s/%s", r.distUrl.String(), docPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *Registry) getRemoteProviderList() (io.ReadCloser, error) {

	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(fmt.Sprintf("%s/%s", r.distUrl.String(), remoteProviderListPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *Registry) getLocalDocRoot() string {
	switch r.localSrcPrefix {
	case "":
		return r.localDocRoot
	default:
		return path.Join(r.localDocRoot, r.localSrcPrefix)
	}
}

func (r *Registry) extractEmbeddedDocs() string {
	switch r.localSrcPrefix {
	case "":
		return r.localDocRoot
	default:
		return path.Join(r.localDocRoot, r.localSrcPrefix)
	}
}

func (r *Registry) listLocalProviders() map[string]struct{} {
	dr := r.getLocalDocRoot()
	switch dr {
	case "":
		return map[string]struct{}{}
	default:
		provs, err := os.ReadDir(dr)
		if err != nil {
			return map[string]struct{}{}
		}
		rv := make(map[string]struct{}, len(provs))
		for _, p := range provs {
			if !strings.HasPrefix(p.Name(), ".") {
				rv[p.Name()] = struct{}{}
			}
		}
		return rv
	}
}

func (r *Registry) getLocalArchiveRoot() string {
	switch r.localDistPrefix {
	case "":
		return r.localDocRoot
	default:
		return path.Join(r.localDocRoot, r.localDistPrefix)
	}
}

func (r *Registry) getLocalDocPath(docPath string) string {
	return path.Join(r.getLocalDocRoot(), docPath)
}

func (r *Registry) getLocalArchivePath(docPath string) string {
	return path.Join(r.getLocalArchiveRoot(), docPath)
}

func (r *Registry) getLocalDoc(docPath string) (io.ReadCloser, error) {
	// localPath := r.getLocalDocPath(docPath)
	fi, err := os.Open(docPath)
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

func (r *Registry) getUnVerifiedArchive(docPath string) (io.ReadCloser, error) {
	if r.isLocalFile() {
		return os.Open(path.Join(r.distUrl.Path, docPath))
	}
	if r.localDocRoot != "" {
		localPath := r.getLocalArchivePath(docPath)
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
		return r.getRemoteArchive(docPath)
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}

func (r *Registry) getEmbeddedVerifiedDocResponse(docPath string) (*edcrypto.VerifierResponse, error) {
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

func (r *Registry) getVerifiedDocResponse(docPath string) (*edcrypto.VerifierResponse, error) {
	var embeddedErr error
	if r.isLocalFile() {
		rb, err := os.Open(path.Join(r.srcUrl.Path, docPath))
		if err != nil {
			return nil, fmt.Errorf("cannot read local registry file: '%s'", err.Error())
		}
		if r.nopVerifier {
			rv := edcrypto.NewVerifierResponse(true, nil, rb, nil)
			return &rv, nil
		}
		sb, err := os.Open(path.Join(r.srcUrl.Path, fmt.Sprintf("%s.sig", docPath)))
		if err != nil {
			return nil, fmt.Errorf("cannot read local signature file: '%s'", err.Error())
		}
		return r.checkSignature(docPath, rb, sb)
	}
	if r.localDocRoot != "" {
		localPath := r.getLocalDocPath(docPath)
		lf, err := r.getLocalDoc(localPath)
		if err == nil {
			if r.nopVerifier {
				rv := edcrypto.NewVerifierResponse(true, nil, lf, nil)
				return &rv, nil
			}
			sf, err := r.getLocalDoc(fmt.Sprintf("%s.sig", localPath))
			if err != nil {
				if lf != nil {
					lf.Close()
				}
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
		if !r.allowSrcDownload {
			return nil, fmt.Errorf("download of individual docs disallowed; please attempt to pull provider docs")
		}
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
		if r.nopVerifier {
			rv := edcrypto.NewVerifierResponse(true, nil, response, nil)
			return &rv, nil
		}
		sigResponse, err := r.getRemoteDoc(fmt.Sprintf("%s.sig", docPath))
		if err != nil {
			response.Close()
			return nil, fmt.Errorf("remote document '%s' present but signature file not present", verifyUrl)
		}
		return r.checkSignature(verifyUrl, response, sigResponse)
	}
	if embeddedErr != nil {
		return nil, fmt.Errorf("error retrieving from embedded: %s", embeddedErr.Error())
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}

func (r *Registry) getVerifiedDocBytes(docPath string) ([]byte, error) {
	vr, err := r.getVerifiedDocResponse(docPath)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(vr.VerifyFile)
}
