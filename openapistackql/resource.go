package openapistackql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-openapi/jsonpointer"
	"vitess.io/vitess/go/sqltypes"
)

type ResourceRegister struct {
	ServiceDocPath  *ServiceRef          `json:"serviceDoc,omitempty" yaml:"serviceDoc,omitempty"`
	Resources       map[string]*Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
	ProviderService *ProviderService     `json:"-" yaml:"-"` // upwards traversal
	Provider        *Provider            `json:"-" yaml:"-"` // upwards traversal
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
	BaseUrl           string                         `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"` // hack
	StackQLConfig     *StackQLConfig                 `json:"config,omitempty" yaml:"config,omitempty"`
	Service           *Service                       `json:"-" yaml:"-"` // upwards traversal
	ProviderService   *ProviderService               `json:"-" yaml:"-"` // upwards traversal
	Provider          *Provider                      `json:"-" yaml:"-"` // upwards traversal
}

func (r *Resource) GetQueryTransposeAlgorithm() string {
	if r.StackQLConfig == nil || r.StackQLConfig.QueryTranspose == nil {
		return ""
	}
	return r.StackQLConfig.QueryTranspose.Algorithm
}

func (r *Resource) GetRequestTranslateAlgorithm() string {
	if r.StackQLConfig == nil || r.StackQLConfig.RequestTranslate == nil {
		return ""
	}
	return r.StackQLConfig.RequestTranslate.Algorithm
}

func (r *Resource) GetPaginationRequestTokenSemantic() (*TokenSemantic, bool) {
	if r.StackQLConfig == nil || r.StackQLConfig.Pagination == nil || r.StackQLConfig.Pagination.RequestToken == nil {
		return nil, false
	}
	return r.StackQLConfig.Pagination.RequestToken, true
}

func (r *Resource) GetPaginationResponseTokenSemantic() (*TokenSemantic, bool) {
	if r.StackQLConfig == nil || r.StackQLConfig.Pagination == nil || r.StackQLConfig.Pagination.ResponseToken == nil {
		return nil, false
	}
	return r.StackQLConfig.Pagination.ResponseToken, true
}

var _ jsonpointer.JSONPointable = (Resource)(Resource{})

func (rsc Resource) JSONLookup(token string) (interface{}, error) {
	if rsc.Methods == nil {
		return nil, fmt.Errorf("Provider.JSONLookup() failure due to prov.ProviderServices == nil")
	}
	ss := strings.Split(token, "/")
	if len(ss) > 1 && ss[len(ss)-2] == "methods" {
		m, ok := rsc.Methods[ss[len(ss)-1]]
		if !ok {
			return nil, fmt.Errorf("cannot resolve json pointer path '%s'", token)
		}
		return &m, nil
	}
	return nil, fmt.Errorf("cannot resolve json pointer path '%s'", token)
}

type MethodSet []*OperationStore

func (ms MethodSet) GetFirstMatch(params map[string]interface{}) (*OperationStore, map[string]interface{}, bool) {
	return ms.getFirstMatch(params)
}

func (ms MethodSet) GetFirst() (*OperationStore, string, bool) {
	return ms.getFirst()
}

func (ms MethodSet) getFirstMatch(params map[string]interface{}) (*OperationStore, map[string]interface{}, bool) {
	for _, m := range ms {
		if remainingParams, ok := m.ParameterMatch(params); ok {
			return m, remainingParams, true
		}
	}
	return nil, params, false
}

func (ms MethodSet) getFirst() (*OperationStore, string, bool) {
	for _, m := range ms {
		return m, m.getName(), true
	}
	return nil, "", false
}

func (rs *Resource) GetDefaultMethodKeysForSQLVerb(sqlVerb string) []string {
	return rs.getDefaultMethodKeysForSQLVerb(sqlVerb)
}

func (rs *Resource) GetMethodsMatched() Methods {
	return rs.getMethodsMatched()
}

func (rs *Resource) matchSQLVerbs() {
	for k, v := range rs.SQLVerbs {
		for _, or := range v {
			orp := &or
			mutated, err := resolveSQLVerbFromResource(rs, orp, k)
			if err == nil && mutated != nil {
				mk := or.extractMethodItem()
				_, ok := rs.Methods[mk]
				if mk != "" && ok {
					rs.Methods[mk] = *mutated
				}
			}
		}
	}
}

func (rs *Resource) getMethodsMatched() Methods {
	rs.matchSQLVerbs()
	rv := rs.Methods
	for k, v := range rv {
		m := v
		sqlVerb := m.SQLVerb
		if sqlVerb == "" {
			sqlVerb = rs.getDefaultSQLVerbForMethodKey(k)
		}
		m.SQLVerb = sqlVerb
		rv[k] = m
	}
	return rv
}

func (rs *Resource) GetFirstMethodMatchFromSQLVerb(sqlVerb string, parameters map[string]interface{}) (*OperationStore, map[string]interface{}, bool) {
	return rs.getFirstMethodMatchFromSQLVerb(sqlVerb, parameters)
}

func (rs *Resource) getFirstMethodMatchFromSQLVerb(sqlVerb string, parameters map[string]interface{}) (*OperationStore, map[string]interface{}, bool) {
	ms, err := rs.getMethodsForSQLVerb(sqlVerb)
	if err != nil {
		return nil, parameters, false
	}
	return ms.getFirstMatch(parameters)
}

func (rs *Resource) GetFirstMethodFromSQLVerb(sqlVerb string) (*OperationStore, string, bool) {
	return rs.getFirstMethodFromSQLVerb(sqlVerb)
}

func (rs *Resource) getUnionRequiredParameters(method *OperationStore) (map[string]Addressable, error) {
	targetSchema, _, err := method.GetSelectSchemaAndObjectPath()
	if err != nil {
		return nil, fmt.Errorf("getUnionRequiredParameters(): cannot infer fat required parameters: %s", err.Error())
	}
	if targetSchema == nil {
		return nil, fmt.Errorf("getUnionRequiredParameters(): target schem is nil")
	}
	targetPath := targetSchema.GetPath()
	rv := make(map[string]Addressable)
	for _, m := range rs.Methods {
		s, _, err := m.GetSelectSchemaAndObjectPath()
		if err != nil || s == nil {
			continue
		}
		methodSchemaPath := s.GetPath()
		if err == nil && s != nil && methodSchemaPath != "" && methodSchemaPath == targetPath {
			reqParams := m.getRequiredParameters()
			for k, v := range reqParams {
				existingParam, ok := rv[k]
				if ok && v.GetType() == existingParam.GetType() {
					return nil, fmt.Errorf("getUnionRequiredParameters(): required params '%s' of conflicting types on resource = '%s'", k, rs.GetName())
				}
				reqParams[k] = v
			}
		}
	}
	return rv, nil
}

func (rs *Resource) getFirstMethodFromSQLVerb(sqlVerb string) (*OperationStore, string, bool) {
	ms, err := rs.getMethodsForSQLVerb(sqlVerb)
	if err != nil {
		return nil, "", false
	}
	return ms.getFirst()
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

func (rs *Resource) getDefaultSQLVerbForMethodKey(methodName string) string {
	switch strings.ToLower(methodName) {
	case "insert", "create":
		return "insert"
	case "delete":
		return "delete"
	case "select", "list", "aggregatedList", "get":
		return "select"
	default:
		return ""
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
		sc, _, err := m.getResponseBodySchemaAndMediaType()
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

func (rs *Resource) GetRequiredParameters() map[string]Addressable {
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
