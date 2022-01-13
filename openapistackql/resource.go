package openapistackql

import (
	"fmt"
	"reflect"

	"vitess.io/vitess/go/sqltypes"
)

type Resource struct {
	ID                string  `json:"id" yaml:"id"`       // Required
	Name              string  `json:"name" yaml:"name"`   // Required
	Title             string  `json:"title" yaml:"title"` // Required
	Description       string  `json:"description,omitempty" yaml:"desription,omitempty"`
	SelectorAlgorithm string  `json:"selectorAlgorithm,omitempty" yaml:"selectorAlgorithm,omitempty"`
	Methods           Methods `json:"methods" yaml:"methods"`

	// Hacks
	BaseUrl string `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
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
	retVal["type"] = rs.GetSelectableObject()
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
