package openapistackql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-openapi/jsonpointer"
	"github.com/stackql/stackql-parser/go/sqltypes"
)

var (
	_ Resource                  = &standardResource{}
	_ jsonpointer.JSONPointable = standardResource{}
)

type Resource interface {
	GetQueryTransposeAlgorithm() string
	GetID() string
	GetName() string
	GetTitle() string
	GetDescription() string
	GetSelectorAlgorithm() string
	GetMethods() Methods
	GetServiceDocPath() *ServiceRef
	GetRequestTranslateAlgorithm() string
	GetPaginationRequestTokenSemantic() (TokenSemantic, bool)
	GetPaginationResponseTokenSemantic() (TokenSemantic, bool)
	FindMethod(key string) (OperationStore, error)
	GetFirstMethodFromSQLVerb(sqlVerb string) (OperationStore, string, bool)
	GetFirstMethodMatchFromSQLVerb(sqlVerb string, parameters map[string]interface{}) (OperationStore, map[string]interface{}, bool)
	GetService() (Service, bool)
	GetViewBodyDDLForSQLDialect(sqlDialect string) (string, bool)
	//
	// unexported mutators
	getSQLVerbs() map[string][]OperationStoreRef
	setProvider(p Provider)
	setService(s Service)
	setProviderService(ps ProviderService)
	getUnionRequiredParameters(method OperationStore) (map[string]Addressable, error)
	setMethod(string, *standardOperationStore)
	mutateSQLVerb(k string, idx int, v OperationStoreRef)
}

type standardResource struct {
	ID                string                         `json:"id" yaml:"id"`       // Required
	Name              string                         `json:"name" yaml:"name"`   // Required
	Title             string                         `json:"title" yaml:"title"` // Required
	Description       string                         `json:"description,omitempty" yaml:"desription,omitempty"`
	SelectorAlgorithm string                         `json:"selectorAlgorithm,omitempty" yaml:"selectorAlgorithm,omitempty"`
	Methods           Methods                        `json:"methods" yaml:"methods"`
	ServiceDocPath    *ServiceRef                    `json:"serviceDoc,omitempty" yaml:"serviceDoc,omitempty"`
	SQLVerbs          map[string][]OperationStoreRef `json:"sqlVerbs" yaml:"sqlVerbs"`
	BaseUrl           string                         `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"` // hack
	StackQLConfig     *standardStackQLConfig         `json:"config,omitempty" yaml:"config,omitempty"`
	Service           Service                        `json:"-" yaml:"-"` // upwards traversal
	ProviderService   ProviderService                `json:"-" yaml:"-"` // upwards traversal
	Provider          Provider                       `json:"-" yaml:"-"` // upwards traversal
}

func (r *standardResource) GetService() (Service, bool) {
	if r.Service == nil {
		return nil, false
	}
	return r.Service, true
}

func (r *standardResource) getSQLVerbs() map[string][]OperationStoreRef {
	return r.SQLVerbs
}

func (r *standardResource) setService(s Service) {
	r.Service = s
}

func (r *standardResource) mutateSQLVerb(k string, idx int, v OperationStoreRef) {
	r.SQLVerbs[k][idx] = v
}

func (r *standardResource) setMethod(k string, v *standardOperationStore) {
	if v == nil {
		return
	}
	r.Methods[k] = *v
}

func (r *standardResource) setProvider(p Provider) {
	r.Provider = p
}

func (r *standardResource) setProviderService(ps ProviderService) {
	r.ProviderService = ps
}

func (r *standardResource) GetID() string {
	return r.ID
}

func (r *standardResource) GetTitle() string {
	return r.Title
}

func (r *standardResource) GetDescription() string {
	return r.Description
}

func (r *standardResource) GetSelectorAlgorithm() string {
	return r.SelectorAlgorithm
}

func (r *standardResource) GetMethods() Methods {
	return r.Methods
}

func (r *standardResource) GetServiceDocPath() *ServiceRef {
	return r.ServiceDocPath
}

func (r *standardResource) GetQueryTransposeAlgorithm() string {
	if r.StackQLConfig == nil || r.StackQLConfig.GetQueryTranspose() == nil {
		return ""
	}
	return r.StackQLConfig.QueryTranspose.Algorithm
}

func (r *standardResource) GetRequestTranslateAlgorithm() string {
	if r.StackQLConfig == nil || r.StackQLConfig.RequestTranslate == nil {
		return ""
	}
	return r.StackQLConfig.RequestTranslate.Algorithm
}

func (r *standardResource) GetPaginationRequestTokenSemantic() (TokenSemantic, bool) {
	if r.StackQLConfig == nil || r.StackQLConfig.GetPagination() == nil || r.StackQLConfig.GetPagination().GetRequestToken() == nil {
		return nil, false
	}
	return r.StackQLConfig.GetPagination().GetRequestToken(), true
}

func (r *standardResource) GetViewBodyDDLForSQLDialect(sqlDialect string) (string, bool) {
	if r.StackQLConfig != nil {
		return r.StackQLConfig.GetViewBodyDDLForSQLDialect(sqlDialect, ViewKeyResourceLevelSelect)
	}
	return "", false
}

func (r *standardResource) GetPaginationResponseTokenSemantic() (TokenSemantic, bool) {
	if r.StackQLConfig == nil || r.StackQLConfig.GetPagination() == nil || r.StackQLConfig.GetPagination().GetResponseToken() == nil {
		return nil, false
	}
	return r.StackQLConfig.GetPagination().GetResponseToken(), true
}

func (rsc standardResource) JSONLookup(token string) (interface{}, error) {
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

func (rs *standardResource) GetDefaultMethodKeysForSQLVerb(sqlVerb string) []string {
	return rs.getDefaultMethodKeysForSQLVerb(sqlVerb)
}

func (rs *standardResource) GetMethodsMatched() Methods {
	return rs.getMethodsMatched()
}

func (rs *standardResource) matchSQLVerbs() {
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

func (rs *standardResource) getMethodsMatched() Methods {
	rs.matchSQLVerbs()
	rv := rs.Methods
	for k, v := range rv {
		m := v
		sqlVerb := m.GetSQLVerb()
		if sqlVerb == "" {
			sqlVerb = rs.getDefaultSQLVerbForMethodKey(k)
		}
		m.setSQLVerb(sqlVerb)
		rv[k] = m
	}
	return rv
}

func (rs *standardResource) GetFirstMethodMatchFromSQLVerb(sqlVerb string, parameters map[string]interface{}) (OperationStore, map[string]interface{}, bool) {
	return rs.getFirstMethodMatchFromSQLVerb(sqlVerb, parameters)
}

func (rs *standardResource) getFirstMethodMatchFromSQLVerb(sqlVerb string, parameters map[string]interface{}) (OperationStore, map[string]interface{}, bool) {
	ms, err := rs.getMethodsForSQLVerb(sqlVerb)
	if err != nil {
		return nil, parameters, false
	}
	return ms.getFirstMatch(parameters)
}

func (rs *standardResource) GetFirstMethodFromSQLVerb(sqlVerb string) (OperationStore, string, bool) {
	return rs.getFirstMethodFromSQLVerb(sqlVerb)
}

func (rs *standardResource) getUnionRequiredParameters(method OperationStore) (map[string]Addressable, error) {
	targetSchema, _, err := method.GetSelectSchemaAndObjectPath()
	if err != nil {
		return nil, fmt.Errorf("getUnionRequiredParameters(): cannot infer fat required parameters: %s", err.Error())
	}
	if targetSchema == nil {
		return nil, fmt.Errorf("getUnionRequiredParameters(): target schem is nil")
	}
	targetPath := targetSchema.GetPath()
	rv := method.getRequiredParameters()
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
				if ok && v.GetType() != existingParam.GetType() {
					return nil, fmt.Errorf("getUnionRequiredParameters(): required params '%s' of conflicting types on resource = '%s'", k, rs.GetName())
				}
				rv[k] = v
			}
		}
	}
	return rv, nil
}

func (rs *standardResource) getFirstMethodFromSQLVerb(sqlVerb string) (OperationStore, string, bool) {
	ms, err := rs.getMethodsForSQLVerb(sqlVerb)
	if err != nil {
		return nil, "", false
	}
	return ms.getFirst()
}

func (rs *standardResource) getDefaultMethodKeysForSQLVerb(sqlVerb string) []string {
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

func (rs *standardResource) getDefaultSQLVerbForMethodKey(methodName string) string {
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

func (rs *standardResource) getMethodsForSQLVerb(sqlVerb string) (MethodSet, error) {
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

func (rs *standardResource) GetSelectableObject() string {
	if m, ok := rs.Methods["list"]; ok {
		sc, _, err := m.getResponseBodySchemaAndMediaType()
		if err == nil {
			return sc.GetName()
		}
	}
	return ""
}

func (rs *standardResource) FindOperationStore(sel OperationSelector) (OperationStore, error) {
	switch rs.SelectorAlgorithm {
	case "", "standard":
		return rs.findOperationStoreStandard(sel)
	}
	return nil, fmt.Errorf("cannot search for operation with selector algorithm = '%s'", rs.SelectorAlgorithm)
}

func (rs *standardResource) findOperationStoreStandard(sel OperationSelector) (OperationStore, error) {
	rv, err := rs.Methods.FindFromSelector(sel)
	if err == nil {
		return rv, nil
	}
	return nil, fmt.Errorf("could not locate operation for resource = %s and sql verb  = %s", rs.Name, sel.GetSQLVerb())
}

func (r *standardResource) ConditionIsValid(lhs string, rhs interface{}) bool {
	elem := r.ToMap(true)[lhs]
	return reflect.TypeOf(elem) == reflect.TypeOf(rhs)
}

func (r *standardResource) FilterBy(predicate func(interface{}) (ITable, error)) (ITable, error) {
	return predicate(r)
}

func (r *standardResource) FindMethod(key string) (OperationStore, error) {
	if r.Methods == nil {
		return nil, fmt.Errorf("cannot find method with key = '%s' from nil methods", key)
	}
	return r.Methods.FindMethod(key)
}

func (rs *standardResource) ToMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	retVal["id"] = rs.ID
	retVal["name"] = rs.Name
	retVal["title"] = rs.Title
	retVal["description"] = rs.Description
	return retVal
}

func (rs *standardResource) GetKeyAsSqlVal(lhs string) (sqltypes.Value, error) {
	val, ok := rs.ToMap(true)[lhs]
	rv, err := InterfaceToSQLType(val)
	if !ok {
		return rv, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return rv, err
}

func (rs *standardResource) GetKey(lhs string) (interface{}, error) {
	val, ok := rs.ToMap(true)[lhs]
	if !ok {
		return nil, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return val, nil
}

func (rs *standardResource) KeyExists(lhs string) bool {
	_, ok := rs.ToMap(true)[lhs]
	return ok
}

func (rs *standardResource) GetRequiredParameters() map[string]Addressable {
	return nil
}

func (rs *standardResource) GetName() string {
	return rs.Name
}

func ResourceConditionIsValid(lhs string, rhs interface{}) bool {
	rs := &standardResource{}
	return rs.ConditionIsValid(lhs, rhs)
}

func ResourceKeyExists(key string) bool {
	rs := &standardResource{}
	return rs.KeyExists(key)
}
