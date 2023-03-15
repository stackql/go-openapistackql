package openapistackql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stackql/stackql-parser/go/sqltypes"
	yaml "gopkg.in/yaml.v3"
)

var (
	_ Service = &standardService{}
)

type Service interface {
	GetT() *openapi3.T
	GetQueryTransposeAlgorithm() string
	IsPreferred() bool
	GetRequestTranslateAlgorithm() string
	GetPaginationRequestTokenSemantic() (TokenSemantic, bool)
	GetPaginationResponseTokenSemantic() (TokenSemantic, bool)
	GetServers() []*openapi3.Server
	GetResources() (map[string]Resource, error)
	GetComponents() openapi3.Components
	GetName() string
	GetResource(resourceName string) (Resource, error)
	GetSchema(key string) (Schema, error)
	GetContactURL() string
	//
	iDiscoveryDoc()
	isObjectSchemaImplicitlyUnioned() bool
	getExtension(key string) (interface{}, bool)
	setStackQLConfig(config StackQLConfig)
	setResourceMap(rsc map[string]*standardResource)
	setProvider(provider Provider)
	getProvider() Provider
	getProviderService() ProviderService
	setProviderService(providerService ProviderService)
	getPath(k string) (*openapi3.PathItem, bool)
}

type standardService struct {
	*openapi3.T
	rsc             map[string]*standardResource
	StackQLConfig   StackQLConfig   `json:"-" yaml:"-"`
	ProviderService ProviderService `json:"-" yaml:"-"` // upwards traversal
	Provider        Provider        `json:"-" yaml:"-"` // upwards traversal
}

func (sv *standardService) GetContactURL() string {
	if sv.Info == nil || sv.Info.Contact == nil {
		return ""
	}
	return sv.Info.Contact.URL
}

func (sv *standardService) getPath(k string) (*openapi3.PathItem, bool) {
	rv, ok := sv.T.Paths[k]
	return rv, ok
}

func (sv *standardService) getProviderService() ProviderService {
	return sv.ProviderService
}

func (sv *standardService) getProvider() Provider {
	return sv.Provider
}

func (sv *standardService) GetComponents() openapi3.Components {
	return sv.T.Components
}

func (sv *standardService) setProvider(provider Provider) {
	sv.Provider = provider
}

func (sv *standardService) setProviderService(providerService ProviderService) {
	sv.ProviderService = providerService
}

func (sv *standardService) setStackQLConfig(config StackQLConfig) {
	sv.StackQLConfig = config
}

func (sv *standardService) getExtension(key string) (interface{}, bool) {
	rv, ok := sv.T.Extensions[key]
	return rv, ok
}

func (sv *standardService) setResourceMap(rsc map[string]*standardResource) {
	sv.rsc = rsc
}

func (sv *standardService) iDiscoveryDoc() {}

func (sv *standardService) GetT() *openapi3.T {
	return sv.T
}

func (sv *standardService) isObjectSchemaImplicitlyUnioned() bool {
	if sv.StackQLConfig != nil {
		return sv.StackQLConfig.isObjectSchemaImplicitlyUnioned()
	}
	if sv.Provider == nil {
		return false
	}
	return sv.Provider.isObjectSchemaImplicitlyUnioned()
}

func NewService(t *openapi3.T) Service {
	svc := &standardService{
		T:   t,
		rsc: make(map[string]*standardResource),
	}
	return svc
}

func (svc *standardService) GetServers() []*openapi3.Server {
	return svc.T.Servers
}

func (svc *standardService) IsPreferred() bool {
	return false
}

func (svc *standardService) GetQueryTransposeAlgorithm() string {
	if svc.StackQLConfig != nil {
		qt, qtExists := svc.StackQLConfig.GetQueryTranspose()
		if qtExists {
			qt.GetAlgorithm()
		}
	}
	return ""
}

func (svc *standardService) GetRequestTranslateAlgorithm() string {
	if svc.StackQLConfig != nil {
		rt, rtExists := svc.StackQLConfig.GetRequestTranslate()
		if rtExists {
			return rt.GetAlgorithm()
		}
	}
	return ""
}

func (svc *standardService) GetPaginationRequestTokenSemantic() (TokenSemantic, bool) {
	if svc.StackQLConfig != nil {
		pag, pagExists := svc.StackQLConfig.GetPagination()
		if pagExists && pag.GetRequestToken() != nil {
			return pag.GetRequestToken(), true
		}
	}
	return nil, false
}

func (svc *standardService) GetPaginationResponseTokenSemantic() (TokenSemantic, bool) {
	if svc.StackQLConfig != nil {
		pag, pagExists := svc.StackQLConfig.GetPagination()
		if pagExists && pag.GetResponseToken() != nil {
			return pag.GetResponseToken(), true
		}
	}
	return nil, false
}

func (svc *standardService) GetSchemas() (map[string]Schema, error) {
	rv := make(map[string]Schema)
	for k, sv := range svc.Components.Schemas {
		rv[k] = NewSchema(sv.Value, svc, k, sv.Ref)
	}
	return rv, nil
}

func (svc *standardService) GetSchema(key string) (Schema, error) {
	svcName := svc.Info.Title
	responseSref, ok := svc.Components.Schemas[key]
	if !ok {
		return nil, fmt.Errorf("cannot find schema for key = '%s' in service title = '%s'", key, svcName)
	}
	responseSchema := responseSref.Value
	if responseSchema == nil {
		return nil, fmt.Errorf("cannot find schema for key = '%s' in service title = '%s'", key, svcName)
	}
	return NewSchema(responseSchema, svc, key, responseSref.Ref), nil
}

func extractExtensionValBytes(extMap map[string]interface{}, key string) ([]byte, error) {
	val, ok := extMap[key]
	if ok {
		switch val := val.(type) {
		case json.RawMessage:
			return val.MarshalJSON()
		default:
			return yaml.Marshal(val)
		}
	}
	return nil, fmt.Errorf("could not find extension key = '%s'", key)
}

func (svc *standardService) GetName() string {
	if sn, err := extractExtensionValBytes(svc.Info.Extensions, "x-serviceName"); err == nil {
		return strings.Trim(string(sn), `"`)
	}
	return svc.Info.Title
}

func (svc *standardService) ToMap() map[string]interface{} {
	retVal := make(map[string]interface{})
	if svc.Info == nil {
		svc.Info = &openapi3.Info{}
	}
	retVal["id"] = svc.Info.Title + ":" + svc.Info.Version
	retVal["name"] = svc.Info.Title
	retVal["title"] = svc.Info.Title
	retVal["description"] = svc.Info.Description
	retVal["version"] = svc.Info.Version
	return retVal
}

func (sv *standardService) KeyExists(lhs string) bool {
	_, ok := sv.ToMap()[lhs]
	return ok
}

func (sv *standardService) GetKeyAsSqlVal(lhs string) (sqltypes.Value, error) {
	val, ok := sv.ToMap()[lhs]
	rv, err := InterfaceToSQLType(val)
	if !ok {
		return rv, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return rv, err
}

func (rs *standardService) GetRequiredParameters() map[string]Addressable {
	return nil
}

func (sv *standardService) GetKey(lhs string) (interface{}, error) {
	val, ok := sv.ToMap()[lhs]
	if !ok {
		return nil, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return val, nil
}

func (sv *standardService) FilterBy(predicate func(interface{}) (ITable, error)) (ITable, error) {
	return predicate(sv)
}

func ServiceKeyExists(key string) bool {
	sv := &standardProviderService{}
	return sv.KeyExists(key)
}

func (sv *standardService) ConditionIsValid(lhs string, rhs interface{}) bool {
	elem := sv.ToMap()[lhs]
	return reflect.TypeOf(elem) == reflect.TypeOf(rhs)
}

func (svc *standardService) GetResources() (map[string]Resource, error) {
	rv := make(map[string]Resource, len(svc.rsc))
	for k, v := range svc.rsc {
		rv[k] = v
	}
	return rv, nil
}

func (svc *standardService) GetResource(resourceName string) (Resource, error) {
	rscs, err := svc.GetResources()
	if err != nil {
		return nil, err
	}
	rsc, ok := rscs[resourceName]
	if !ok {
		return nil, fmt.Errorf("Service.GetResource() failure")
	}
	return rsc, nil
}

func ServiceConditionIsValid(lhs string, rhs interface{}) bool {
	sv := &standardProviderService{}
	return sv.ConditionIsValid(lhs, rhs)
}
