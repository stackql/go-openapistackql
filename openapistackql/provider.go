package openapistackql

import (
	"fmt"
	"reflect"

	"github.com/getkin/kin-openapi/jsoninfo"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	"vitess.io/vitess/go/sqltypes"
)

type ResponseKeys struct {
	SelectItemsKey string `json:"selectItemsKey,omitempty" yaml:"selectItemsKey,omitempty"`
	DeleteItemsKey string `json:"deleteItemsKey,omitempty" yaml:"deleteItemsKey,omitempty"`
}

type Provider struct {
	openapi3.ExtensionProps
	ResponseKeys
	ID               string                     `json:"id" yaml:"id"`
	Name             string                     `json:"name" yaml:"name"`
	Title            string                     `json:"title" yaml:"title"`
	Version          string                     `json:"version" yaml:"version"`
	Description      string                     `json:"description,omitempty" yaml:"desription,omitempty"`
	ProviderServices map[string]ProviderService `json:"providerServices,omitempty" yaml:"providerServices,omitempty"`
}

type ProviderService struct {
	openapi3.ExtensionProps
	ID          string      `json:"id" yaml:"id"`           // Required
	Name        string      `json:"name" yaml:"name"`       // Required
	Title       string      `json:"title" yaml:"title"`     // Required
	Version     string      `json:"version" yaml:"version"` // Required
	Description string      `json:"description" yaml:"description"`
	Preferred   bool        `json:"preferred" yaml:"preferred"`
	ServiceRef  *ServiceRef `json:"service" yaml:"service"` // will be lazy evaluated
}

func (sv *ProviderService) ConditionIsValid(lhs string, rhs interface{}) bool {
	elem := sv.ToMap()[lhs]
	return reflect.TypeOf(elem) == reflect.TypeOf(rhs)
}

func getService(url string) (*Service, error) {
	b, err := GetServiceDocBytes(url)
	if err != nil {
		return nil, err
	}
	return LoadServiceDocFromBytes(b)
}

func (pr *Provider) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(pr)
}

func (pr *Provider) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, pr)
}

func (pr *ProviderService) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(pr)
}

func (pr *ProviderService) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, pr)
}

func (ps *ProviderService) FilterBy(predicate func(interface{}) (ITable, error)) (ITable, error) {
	return predicate(ps)
}

func (ps *ProviderService) ToMap() map[string]interface{} {
	retVal := make(map[string]interface{})
	retVal["id"] = ps.ID
	retVal["name"] = ps.Name
	retVal["title"] = ps.Title
	retVal["description"] = ps.Description
	retVal["version"] = ps.Version
	return retVal
}

func (ps *ProviderService) GetKeyAsSqlVal(lhs string) (sqltypes.Value, error) {
	val, ok := ps.ToMap()[lhs]
	rv, err := InterfaceToSQLType(val)
	if !ok {
		return rv, fmt.Errorf("key '%s' no preset in providerService", lhs)
	}
	return rv, err
}

func (ps *ProviderService) GetKey(lhs string) (interface{}, error) {
	val, ok := ps.ToMap()[lhs]
	if !ok {
		return nil, fmt.Errorf("key '%s' no preset in providerService", lhs)
	}
	return val, nil
}

func (pr *Provider) GetService(key string) (*Service, error) {
	sh, ok := pr.ProviderServices[key]
	if !ok {
		return nil, fmt.Errorf("cannot resolve service with key = '%s'", key)
	}
	return sh.GetService()
}

func (ps ProviderService) GetService() (*Service, error) {
	if ps.ServiceRef.Value != nil {
		return ps.ServiceRef.Value, nil
	}
	svc, err := getService(ps.ServiceRef.Ref)
	if err != nil {
		return nil, err
	}
	ps.ServiceRef.Value = svc
	return ps.ServiceRef.Value, nil
}

func (ps *ProviderService) GetName() string {
	return ps.Name
}

func (ps *ProviderService) GetRequiredParameters() map[string]*Parameter {
	return nil
}

func (ps *ProviderService) KeyExists(lhs string) bool {
	_, ok := ps.ToMap()[lhs]
	return ok
}

var _ jsonpointer.JSONPointable = (Provider)(Provider{})

func (prov Provider) JSONLookup(token string) (interface{}, error) {
	if prov.ProviderServices == nil {
		return nil, fmt.Errorf("Provider.JSONLookup() failure due to prov.ProviderServices == nil")
	}
	ps, ok := prov.ProviderServices[token]
	if !ok {
		return nil, fmt.Errorf("Provider.JSONLookup() failure")
	}
	return &ps, nil
}

func NewProvider(id, name, title, version string) *Provider {
	return &Provider{
		ID:      id,
		Name:    name,
		Title:   title,
		Version: version,
	}
}

func (pr *Provider) iDiscoveryDoc() {}

func (pr *Provider) Debug() string {
	return fmt.Sprintf("%v", pr)
}
