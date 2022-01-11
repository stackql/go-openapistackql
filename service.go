package openapistackql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	"vitess.io/vitess/go/sqltypes"
)

type Service struct {
	*openapi3.T
	rsc map[string]*Resource
}

func (sv *Service) iDiscoveryDoc() {}

func (sv *Service) GetT() *openapi3.T {
	return sv.T
}

func NewService(t *openapi3.T) *Service {
	svc := &Service{
		T:   t,
		rsc: make(map[string]*Resource),
	}
	return svc
}

func (svc *Service) IsPreferred() bool {
	return false
}

func (svc *Service) FindRoute(req *http.Request) {
	router, _ := gorillamux.NewRouter(svc.GetT())
	route, pathParams, err := router.FindRoute(req)
	log.Infoln(fmt.Sprintf("route = %v, pathParams =  %v, err = %v", route, pathParams, err))
}

func (svc *Service) GetSchemas() (map[string]*Schema, error) {
	rv := make(map[string]*Schema)
	for k, sv := range svc.Components.Schemas {
		rv[k] = NewSchema(sv.Value, k)
	}
	return rv, nil
}

func (svc *Service) GetSchema(key string) (*Schema, error) {
	svcName := svc.Info.Title
	responseSref, ok := svc.Components.Schemas[key]
	if !ok {
		return nil, fmt.Errorf("cannot find schema for key = '%s' in service title = '%s'", key, svcName)
	}
	responseSchema := responseSref.Value
	if responseSchema == nil {
		return nil, fmt.Errorf("cannot find schema for key = '%s' in service title = '%s'", key, svcName)
	}
	return NewSchema(responseSchema, key), nil
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

func (svc *Service) GetName() string {
	if sn, err := extractExtensionValBytes(svc.Info.Extensions, "x-serviceName"); err == nil {
		return strings.Trim(string(sn), `"`)
	}
	return svc.Info.Title
}

func (svc *Service) ToMap() map[string]interface{} {
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

func (sv *Service) KeyExists(lhs string) bool {
	_, ok := sv.ToMap()[lhs]
	return ok
}

func (sv *Service) GetKeyAsSqlVal(lhs string) (sqltypes.Value, error) {
	val, ok := sv.ToMap()[lhs]
	rv, err := InterfaceToSQLType(val)
	if !ok {
		return rv, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return rv, err
}

func (rs *Service) GetRequiredParameters() map[string]*Parameter {
	return nil
}

func (sv *Service) GetKey(lhs string) (interface{}, error) {
	val, ok := sv.ToMap()[lhs]
	if !ok {
		return nil, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return val, nil
}

func (sv *Service) FilterBy(predicate func(interface{}) (ITable, error)) (ITable, error) {
	return predicate(sv)
}

func ServiceKeyExists(key string) bool {
	sv := ProviderService{}
	return sv.KeyExists(key)
}

func (sv *Service) ConditionIsValid(lhs string, rhs interface{}) bool {
	elem := sv.ToMap()[lhs]
	return reflect.TypeOf(elem) == reflect.TypeOf(rhs)
}

func (svc *Service) GetResources() (map[string]*Resource, error) {
	return svc.rsc, nil
}

func (svc *Service) GetResource(resourceName string) (*Resource, error) {
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
	sv := &ProviderService{}
	return sv.ConditionIsValid(lhs, rhs)
}
