package openapistackql

import (
	"fmt"
	"reflect"
	"strings"

	"vitess.io/vitess/go/sqltypes"
)

type ResourceRegister struct {
	ServiceDocPath *ServiceRef          `json:"serviceDoc,omitempty" yaml:"serviceDoc,omitempty"`
	Resources      map[string]*Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
}

func (rr *ResourceRegister) ObtainServiceDocUrl(resourceKey string) string {
	var rv string
	if rr.ServiceDocPath != nil {
		rv = rr.ServiceDocPath.Ref
	}
	rsc, ok := rr.Resources[resourceKey]
	if ok && rsc.ServiceDocPath != nil && rsc.ServiceDocPath.Ref != "" {
		rv = rsc.ServiceDocPath.Ref
	}
	return rv
}

func NewResourceRegister() *ResourceRegister {
	return &ResourceRegister{
		ServiceDocPath: &ServiceRef{},
		Resources:      make(map[string]*Resource),
	}
}

type Resource struct {
	ID                string                         `json:"id" yaml:"id"`       // Required
	Name              string                         `json:"name" yaml:"name"`   // Required
	Title             string                         `json:"title" yaml:"title"` // Required
	Description       string                         `json:"description,omitempty" yaml:"desription,omitempty"`
	SelectorAlgorithm string                         `json:"selectorAlgorithm,omitempty" yaml:"selectorAlgorithm,omitempty"`
	Methods           Methods                        `json:"methods" yaml:"methods"`
	ServiceDocPath    *ServiceRef                    `json:"serviceDoc,omitempty" yaml:"serviceDoc,omitempty"`
	SQLVerbs          map[string][]OperationStoreRef `json:"sqlVerbs" yaml:"sqlVerbs"`

	// Hacks
	BaseUrl string `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
}

type MethodSet []*OperationStore

func (ms MethodSet) GetFirstMatch(params map[string]interface{}) (*OperationStore, bool) {
	return ms.getFirstMatch(params)
}

func (ms MethodSet) getFirstMatch(params map[string]interface{}) (*OperationStore, bool) {
	for _, m := range ms {
		if m.IsParameterMatch(params) {
			return m, true
		}
	}
	return nil, false
}

func (rs *Resource) GetDefaultMethodKeysForSQLVerb(sqlVerb string) []string {
	return rs.getDefaultMethodKeysForSQLVerb(sqlVerb)
}

func (rs *Resource) getDefaultMethodKeysForSQLVerb(sqlVerb string) []string {
	switch strings.ToLower(sqlVerb) {
	case "insert":
		return []string{"insert", "create"}
	case "delete":
		return []string{"delete"}
	case "select":
		return []string{"select", "list", "aggregatedList", "get"}
	default:
		return []string{}
	}
}

func (rs *Resource) getMethodsForSQLVerb(sqlVerb string) (MethodSet, error) {
	var retVal MethodSet
	v, ok := rs.SQLVerbs[sqlVerb]
	if ok {
		for _, opt := range v {
			if opt.Value != nil {
				retVal = append(retVal, opt.Value)
			}
		}
		if len(retVal) > 0 {
			return retVal, nil
		}
	} else {
		defaultMethodKeys := rs.getDefaultMethodKeysForSQLVerb(sqlVerb)
		for _, k := range defaultMethodKeys {
			m, ok := rs.Methods[k]
			if ok {
				retVal = append(retVal, &m)
			}
		}
		if len(retVal) > 0 {
			return retVal, nil
		}
	}
	return nil, fmt.Errorf("could not resolve SQL verb '%s'", sqlVerb)
}

func (rs *Resource) GetSelectableObject() string {
	if m, ok := rs.Methods["list"]; ok {
		sc, err := m.GetResponseBodySchema()
		if err == nil {
			return sc.GetName()
		}
	}
	return ""
}

func (rs *Resource) FindOperationStore(sel OperationSelector) (*OperationStore, error) {
	switch rs.SelectorAlgorithm {
	case "", "standard":
		return rs.findOperationStoreStandard(sel)
	}
	return nil, fmt.Errorf("cannot search for operation with selector algorithm = '%s'", rs.SelectorAlgorithm)
}

func (rs *Resource) findOperationStoreStandard(sel OperationSelector) (*OperationStore, error) {
	rv, err := rs.Methods.FindFromSelector(sel)
	if err == nil {
		return rv, nil
	}
	return nil, fmt.Errorf("could not locate operation for resource = %s and sql verb  = %s", rs.Name, sel.SQLVerb)
}

func (r *Resource) ConditionIsValid(lhs string, rhs interface{}) bool {
	elem := r.ToMap(true)[lhs]
	return reflect.TypeOf(elem) == reflect.TypeOf(rhs)
}

func (r *Resource) FilterBy(predicate func(interface{}) (ITable, error)) (ITable, error) {
	return predicate(r)
}

func (r *Resource) FindMethod(key string) (*OperationStore, error) {
	if r.Methods == nil {
		return nil, fmt.Errorf("cannot find method with key = '%s' from nil methods", key)
	}
	return r.Methods.FindMethod(key)
}

func (rs *Resource) ToMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	retVal["id"] = rs.ID
	retVal["name"] = rs.Name
	retVal["title"] = rs.Title
	retVal["description"] = rs.Description
	return retVal
}

func (rs *Resource) GetKeyAsSqlVal(lhs string) (sqltypes.Value, error) {
	val, ok := rs.ToMap(true)[lhs]
	rv, err := InterfaceToSQLType(val)
	if !ok {
		return rv, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return rv, err
}

func (rs *Resource) GetKey(lhs string) (interface{}, error) {
	val, ok := rs.ToMap(true)[lhs]
	if !ok {
		return nil, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return val, nil
}

func (rs *Resource) KeyExists(lhs string) bool {
	_, ok := rs.ToMap(true)[lhs]
	return ok
}

func (rs *Resource) GetRequiredParameters() map[string]*Parameter {
	return nil
}

func (rs *Resource) GetName() string {
	return rs.Name
}

func ResourceConditionIsValid(lhs string, rhs interface{}) bool {
	rs := &Resource{}
	return rs.ConditionIsValid(lhs, rhs)
}

func ResourceKeyExists(key string) bool {
	rs := &Resource{}
	return rs.KeyExists(key)
}
