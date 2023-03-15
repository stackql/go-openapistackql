package openapistackql

import (
	"fmt"

	"github.com/getkin/kin-openapi/jsoninfo"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
)

var (
	_ jsonpointer.JSONPointable = (Provider)(&standardProvider{})
)

type ResponseKeys struct {
	SelectItemsKey string `json:"selectItemsKey,omitempty" yaml:"selectItemsKey,omitempty"`
	DeleteItemsKey string `json:"deleteItemsKey,omitempty" yaml:"deleteItemsKey,omitempty"`
}

type Provider interface {
	Debug() string
	GetAuth() (AuthDTO, bool)
	GetDeleteItemsKey() string
	GetName() string
	GetProviderServices() map[string]ProviderService
	GetPaginationRequestTokenSemantic() (TokenSemantic, bool)
	GetPaginationResponseTokenSemantic() (TokenSemantic, bool)
	GetProviderService(key string) (ProviderService, error)
	GetQueryTransposeAlgorithm() string
	GetRequestTranslateAlgorithm() string
	GetResourcesShallow(serviceKey string) (ResourceRegister, error)
	GetStackQLConfig() (StackQLConfig, bool)
	JSONLookup(token string) (interface{}, error)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
	//
	getResourcesShallowWithRegistry(registry RegistryAPI, serviceKey string) (ResourceRegister, error)
	isObjectSchemaImplicitlyUnioned() bool
}

type standardProvider struct {
	openapi3.ExtensionProps
	ResponseKeys
	FilePath         string                              `json:"-" yaml:"-"`
	ID               string                              `json:"id" yaml:"id"`
	Name             string                              `json:"name" yaml:"name"`
	Title            string                              `json:"title" yaml:"title"`
	Version          string                              `json:"version" yaml:"version"`
	Description      string                              `json:"description,omitempty" yaml:"desription,omitempty"`
	ProviderServices map[string]*standardProviderService `json:"providerServices,omitempty" yaml:"providerServices,omitempty"`
	StackQLConfig    *standardStackQLConfig              `json:"config,omitempty" yaml:"config,omitempty"`
}

func (pr *standardProvider) GetAuth() (AuthDTO, bool) {
	if pr.StackQLConfig != nil {
		return pr.StackQLConfig.GetAuth()
	}
	return nil, false
}

func (pr *standardProvider) GetProviderServices() map[string]ProviderService {
	providerServices := make(map[string]ProviderService, len(pr.ProviderServices))
	for k, v := range pr.ProviderServices {
		providerServices[k] = v
	}
	return providerServices
}

func (pr *standardProvider) GetName() string {
	return pr.Name
}

func (pr *standardProvider) GetStackQLConfig() (StackQLConfig, bool) {
	return pr.StackQLConfig, pr.StackQLConfig != nil
}

func (pr *standardProvider) GetDeleteItemsKey() string {
	return pr.DeleteItemsKey
}

func (pr *standardProvider) GetQueryTransposeAlgorithm() string {
	if pr.StackQLConfig == nil || pr.StackQLConfig.QueryTranspose == nil {
		return ""
	}
	return pr.StackQLConfig.QueryTranspose.Algorithm
}

func (pr *standardProvider) GetRequestTranslateAlgorithm() string {
	if pr.StackQLConfig == nil || pr.StackQLConfig.RequestTranslate == nil {
		return ""
	}
	return pr.StackQLConfig.RequestTranslate.Algorithm
}

func (pr *standardProvider) isObjectSchemaImplicitlyUnioned() bool {
	if pr.StackQLConfig != nil {
		return pr.StackQLConfig.isObjectSchemaImplicitlyUnioned()
	}
	return false
}

func (pr *standardProvider) GetPaginationRequestTokenSemantic() (TokenSemantic, bool) {
	if pr.StackQLConfig == nil || pr.StackQLConfig.Pagination == nil || pr.StackQLConfig.Pagination.RequestToken == nil {
		return nil, false
	}
	return pr.StackQLConfig.Pagination.RequestToken, true
}

func (pr *standardProvider) GetPaginationResponseTokenSemantic() (TokenSemantic, bool) {
	if pr.StackQLConfig == nil || pr.StackQLConfig.Pagination == nil || pr.StackQLConfig.Pagination.ResponseToken == nil {
		return nil, false
	}
	return pr.StackQLConfig.Pagination.ResponseToken, true
}

func (pr *standardProvider) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(pr)
}

func (pr *standardProvider) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, pr)
}

func (pr *standardProvider) getServiceWithRegistry(registry RegistryAPI, key string) (Service, error) {
	sh, err := pr.getProviderService(key)
	if err != nil {
		return nil, err
	}
	return sh.getServiceWithRegistry(registry)
}

func (pr *standardProvider) GetService(key string) (Service, error) {
	sh, err := pr.getProviderService(key)
	if err != nil {
		return nil, err
	}
	return sh.GetService()
}

func (pr *standardProvider) getResourcesShallowWithRegistry(registry RegistryAPI, serviceKey string) (ResourceRegister, error) {
	sh, err := pr.getProviderService(serviceKey)
	if err != nil {
		return nil, err
	}
	return sh.getResourcesShallowWithRegistry(registry)
}

func (pr *standardProvider) GetResourcesShallow(serviceKey string) (ResourceRegister, error) {
	sh, err := pr.getProviderService(serviceKey)
	if err != nil {
		return nil, err
	}
	return sh.GetResourcesShallow()
}

func (pr *standardProvider) getProviderService(key string) (ProviderService, error) {
	sh, ok := pr.ProviderServices[key]
	if !ok {
		return nil, fmt.Errorf("cannot resolve service with key = '%s'", key)
	}
	return sh, nil
}

func (pr *standardProvider) GetProviderService(key string) (ProviderService, error) {
	return pr.getProviderService(key)
}

func (prov *standardProvider) JSONLookup(token string) (interface{}, error) {
	if prov.ProviderServices == nil {
		return nil, fmt.Errorf("Provider.JSONLookup() failure due to prov.ProviderServices == nil")
	}
	ps, ok := prov.ProviderServices[token]
	if !ok {
		return nil, fmt.Errorf("Provider.JSONLookup() failure")
	}
	return &ps, nil
}

func NewProvider(id, name, title, version string) Provider {
	return &standardProvider{
		ID:      id,
		Name:    name,
		Title:   title,
		Version: version,
	}
}

func (pr *standardProvider) iDiscoveryDoc() {}

func (pr *standardProvider) Debug() string {
	return fmt.Sprintf("%v", pr)
}
