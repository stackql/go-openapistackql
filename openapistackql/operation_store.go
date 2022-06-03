package openapistackql

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stackql/go-openapistackql/pkg/queryrouter"

	log "github.com/sirupsen/logrus"

	"vitess.io/vitess/go/sqltypes"
)

const (
	defaultSelectItemsKey = "items"
)

type Methods map[string]OperationStore

func (ms Methods) FindMethod(key string) (*OperationStore, error) {
	if m, ok := ms[key]; ok {
		return &m, nil
	}
	return nil, fmt.Errorf("could not find method for key = '%s'", key)
}

func sortOperationStoreSlices(opSlices ...[]OperationStore) {
	for _, opSlice := range opSlices {
		sort.SliceStable(opSlice, func(i, j int) bool {
			return opSlice[i].MethodKey < opSlice[j].MethodKey
		})
	}
}

func (ms Methods) OrderMethods() ([]OperationStore, error) {
	var selectBin, insertBin, deleteBin, updateBin, execBin []OperationStore
	for k, v := range ms {
		switch v.SQLVerb {
		case "select":
			v.MethodKey = k
			selectBin = append(selectBin, v)
		case "insert":
			v.MethodKey = k
			selectBin = append(insertBin, v)
		case "update":
			v.MethodKey = k
			selectBin = append(updateBin, v)
		case "delete":
			v.MethodKey = k
			selectBin = append(deleteBin, v)
		case "exec":
			v.MethodKey = k
			selectBin = append(execBin, v)
		default:
			return nil, fmt.Errorf("cannot accomodate sqlVerb = '%s'", v.SQLVerb)
		}
	}
	sortOperationStoreSlices(selectBin, insertBin, deleteBin, updateBin, execBin)
	selectBin = append(selectBin, append(insertBin, append(deleteBin, append(updateBin, execBin...)...)...)...)
	return selectBin, nil
}

func (ms Methods) FindFromSelector(sel OperationSelector) (*OperationStore, error) {
	for _, m := range ms {
		if m.SQLVerb == sel.SQLVerb {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("could not locate operation for sql verb  = %s", sel.SQLVerb)
}

type OperationSelector struct {
	SQLVerb string `json:"sqlVerb" yaml:"sqlVerb"` // Required
	// Optional parameters.
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

func NewOperationSelector(slqVerb string, params map[string]interface{}) OperationSelector {
	return OperationSelector{
		SQLVerb:    slqVerb,
		Parameters: params,
	}
}

type ExpectedRequest struct {
	BodyMediaType string `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
	Schema        *Schema
	Required      []string `json:"required,omitempty" yaml:"required,omitempty"`
}

type ExpectedResponse struct {
	BodyMediaType string `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
	OpenAPIDocKey string `json:"openAPIDocKey,omitempty" yaml:"openAPIDocKey,omitempty"`
	ObjectKey     string `json:"objectKey,omitempty" yaml:"objectKey,omitempty"`
	Schema        *Schema
}

type OperationStore struct {
	MethodKey string `json:"-" yaml:"-"`
	SQLVerb   string `json:"-" yaml:"-"` // Required
	// Optional parameters.
	Parameters   map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	PathItem     *openapi3.PathItem     `json:"-" yaml:"-"`                 // Required
	APIMethod    string                 `json:"apiMethod" yaml:"apiMethod"` // Required
	OperationRef *OperationRef          `json:"operation" yaml:"operation"` // Required
	PathRef      *PathItemRef           `json:"path" yaml:"path"`           // Deprecated
	Request      *ExpectedRequest       `json:"request" yaml:"request"`
	Response     *ExpectedResponse      `json:"response" yaml:"response"`
	Servers      *openapi3.Servers      `json:"servers" yaml:"servers"`
	// private
	parameterizedPath string `json:"-" yaml:"-"`
}

func (op *OperationStore) ParameterMatch(params map[string]interface{}) (map[string]interface{}, bool) {
	return op.parameterMatch(params)
}

func (op *OperationStore) parameterMatch(params map[string]interface{}) (map[string]interface{}, bool) {
	copiedParams := make(map[string]interface{})
	for k, v := range params {
		copiedParams[k] = v
	}
	requiredParameters := NewParameterSuffixMap()
	optionalParameters := NewParameterSuffixMap()
	for k, v := range op.getRequiredParameters() {
		key := fmt.Sprintf("%s.%s", op.getName(), k)
		_, keyExists := requiredParameters.Get(key)
		if keyExists {
			return copiedParams, false
		}
		requiredParameters.Put(key, v)
	}
	for k, vOpt := range op.getOptionalParameters() {
		key := fmt.Sprintf("%s.%s", op.getName(), k)
		_, keyExists := optionalParameters.Get(key)
		if keyExists {
			return copiedParams, false
		}
		optionalParameters.Put(key, vOpt)
	}
	for k := range copiedParams {
		if requiredParameters.Delete(k) {
			delete(copiedParams, k)
			continue
		}
		if optionalParameters.Delete(k) {
			delete(copiedParams, k)
			continue
		}
		log.Debugf("parameter '%s' unmatched for method '%s'\n", k, op.getName())
	}
	if requiredParameters.Size() == 0 {
		return copiedParams, true
	}
	log.Debugf("unmatched **required** paramter count = %d for method '%s'\n", requiredParameters.Size(), op.getName())
	return copiedParams, false
}

func (op *OperationStore) GetParameterizedPath() string {
	return op.parameterizedPath
}

func (op *OperationStore) GetOptimalResponseMediaType() string {
	return op.getOptimalResponseMediaType()
}

func (op *OperationStore) getOptimalResponseMediaType() string {
	if op.Response != nil && op.Response.BodyMediaType != "" {
		return op.Response.BodyMediaType
	}
	return MediaTypeJson
}

func (op *OperationStore) IsNullary() bool {
	rbs, _, _ := op.GetResponseBodySchemaAndMediaType()
	return rbs == nil
}

func (m *OperationStore) KeyExists(lhs string) bool {
	if lhs == MethodName {
		return true
	}
	if m.OperationRef == nil {
		return false
	}
	if m.OperationRef.Value == nil {
		return false
	}
	params := m.OperationRef.Value.Parameters
	if params == nil {
		return false
	}
	for _, p := range params {
		if p.Value == nil {
			continue
		}
		if lhs == p.Value.Name {
			return true
		}
	}
	if m.Servers != nil {
		for _, s := range *m.Servers {
			for k, _ := range s.Variables {
				if lhs == k {
					return true
				}
			}
		}
	}
	return false
}

func (m *OperationStore) GetSelectItemsKey() string {
	return m.getSelectItemsKeySimple()
}

func (m *OperationStore) getSelectItemsKeySimple() string {
	if m.Response != nil {
		return m.Response.ObjectKey
	}
	return ""
}

func (m *OperationStore) GetKey(lhs string) (interface{}, error) {
	val, ok := m.ToPresentationMap(true)[lhs]
	if !ok {
		return nil, fmt.Errorf("key '%s' no preset in metadata_method", lhs)
	}
	return val, nil
}

func (m *OperationStore) GetColumnOrder(extended bool) []string {
	retVal := []string{
		MethodName,
		RequiredParams,
		SQLVerb,
	}
	if extended {
		retVal = append(retVal, MethodDescription)
	}
	return retVal
}

func (m *OperationStore) IsAwaitable() bool {
	rs, _, err := m.GetResponseBodySchemaAndMediaType()
	if err != nil {
		return false
	}
	return strings.HasSuffix(rs.key, "Operation")
}

func (m *OperationStore) FilterBy(predicate func(interface{}) (ITable, error)) (ITable, error) {
	return predicate(m)
}

func (m *OperationStore) GetKeyAsSqlVal(lhs string) (sqltypes.Value, error) {
	val, ok := m.ToPresentationMap(true)[lhs]
	rv, err := InterfaceToSQLType(val)
	if !ok {
		return rv, fmt.Errorf("key '%s' no preset in metadata_service", lhs)
	}
	return rv, err
}

func (m *OperationStore) GetRequiredParameters() map[string]*Parameter {
	return m.getRequiredParameters()
}

func (m *OperationStore) getRequiredParameters() map[string]*Parameter {
	retVal := make(map[string]*Parameter)
	if m.OperationRef.Value == nil || m.OperationRef.Value.Parameters == nil {
		return retVal
	}
	for _, p := range m.OperationRef.Value.Parameters {
		param := p.Value
		if param != nil && param.Required {
			retVal[param.Name] = (*Parameter)(p.Value)
		}
	}
	return retVal
}

func (m *OperationStore) GetOptionalParameters() map[string]*Parameter {
	return m.getOptionalParameters()
}

func (m *OperationStore) getOptionalParameters() map[string]*Parameter {
	retVal := make(map[string]*Parameter)
	if m.OperationRef == nil || m.OperationRef.Value.Parameters == nil {
		return retVal
	}
	for _, p := range m.OperationRef.Value.Parameters {
		param := p.Value
		if param != nil && !param.Required {
			retVal[param.Name] = (*Parameter)(p.Value)
		}
	}
	return retVal
}

func (ops *OperationStore) getMethod() (*openapi3.Operation, error) {
	if ops.OperationRef != nil && ops.OperationRef.Value != nil {
		return ops.OperationRef.Value, nil
	}
	return nil, fmt.Errorf("no method attached to operation store")
}

func (m *OperationStore) GetParameters() map[string]*Parameter {
	retVal := make(map[string]*Parameter)
	if m.OperationRef == nil || m.OperationRef.Value.Parameters == nil {
		return retVal
	}
	for _, p := range m.OperationRef.Value.Parameters {
		param := p.Value
		if param != nil {
			retVal[param.Name] = (*Parameter)(p.Value)
		}
	}
	return retVal
}

func (m *OperationStore) GetParameter(paramKey string) (*Parameter, bool) {
	params := m.GetParameters()
	rv, ok := params[paramKey]
	return rv, ok
}

func (m *OperationStore) GetName() string {
	return m.getName()
}

func (m *OperationStore) getName() string {
	if m.OperationRef != nil && m.OperationRef.Value != nil && m.OperationRef.Value.OperationID != "" {
		return m.OperationRef.Value.OperationID
	}
	return m.MethodKey
}

func (m *OperationStore) ToPresentationMap(extended bool) map[string]interface{} {
	requiredParams := m.GetRequiredParameters()
	var requiredParamNames []string
	for s := range requiredParams {
		requiredParamNames = append(requiredParamNames, s)
	}
	var requiredBodyParamNames []string
	rs, err := m.GetRequestBodySchema()
	if rs != nil && err == nil {
		for k, pr := range rs.Properties {
			if pr == nil || pr.Value == nil {
				continue
			}
			paramName := fmt.Sprintf("%s%s", RequestBodyBaseKey, k)
			sc := pr.Value
			if rs.IsRequired(k) || m.IsRequiredRequestBodyProperty(k) {
				requiredBodyParamNames = append(requiredBodyParamNames, paramName)
				continue
			}
			for _, methodName := range sc.Required {
				if methodName == m.GetName() {
					requiredBodyParamNames = append(requiredBodyParamNames, paramName)
				}
			}
		}
	}
	sort.Strings(requiredParamNames)
	sort.Strings(requiredBodyParamNames)
	requiredParamNames = append(requiredParamNames, requiredBodyParamNames...)

	sqlVerb := m.SQLVerb
	if sqlVerb == "" {
		sqlVerb = "EXEC"
	}

	retVal := map[string]interface{}{
		MethodName:     m.MethodKey,
		RequiredParams: strings.Join(requiredParamNames, ", "),
		SQLVerb:        strings.ToUpper(sqlVerb),
	}
	if extended {
		retVal[MethodDescription] = m.OperationRef.Value.Description
	}
	return retVal
}

func (op *OperationStore) GetOperationParameters() Parameters {
	return Parameters(op.OperationRef.Value.Parameters)
}

func (op *OperationStore) GetOperationParameter(key string) (*Parameter, bool) {
	params := Parameters(op.OperationRef.Value.Parameters)
	if params == nil {
		return nil, false
	}
	return params.GetParameter(key)
}

func GetServersFromHeirarchy(t *Service, op *OperationStore) openapi3.Servers {
	return getServersFromHeirarchy(t, op)
}

func getServersFromHeirarchy(t *Service, op *OperationStore) openapi3.Servers {
	servers := t.Servers
	if servers == nil || (op.OperationRef.Value.Servers != nil && len(*op.OperationRef.Value.Servers) > 0) {
		servers = *op.OperationRef.Value.Servers
	}
	return servers
}

func selectServer(servers openapi3.Servers, inputParams map[string]interface{}) (string, error) {
	paramsConformed := make(map[string]string)
	for k, v := range inputParams {
		switch v := v.(type) {
		case string:
			paramsConformed[k] = v
		}
	}
	srvs, err := obtainServerURLsFromServers(servers, paramsConformed)
	if err != nil {
		return "", err
	}
	return srvs[0], nil
}

func (op *OperationStore) acceptPathParam(mutableParamMap map[string]interface{}) {}

func marshalBody(body interface{}, contentType string) ([]byte, error) {
	switch contentType {
	case "application/json":
		return json.Marshal(body)
	case "application/xml":
		return xml.Marshal(body)
	}
	return nil, fmt.Errorf("media type = '%s' not supported", contentType)
}

func unmarshalBody(bytes []byte, obj interface{}, contentType string) error {
	switch contentType {
	case "application/json":
		return json.Unmarshal(bytes, obj)
	case "application/xml":
		return xml.Unmarshal(bytes, obj)
	}
	return fmt.Errorf("media type = '%s' not supported", contentType)
}

// func (op *OperationStore) ProcessResponse(body []byte) (interface{}, error) {
// 	switch op.Response.Schema.Type {
// 	case "string": // (this includes dates and files)
// 		return string(body), nil
// 	case "number":
// 		return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
// 	case "integer":
// 		return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
// 	case "boolean":
// 		return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
// 	case "array":
// 		return marshalBody(body, op.Response.BodyMediaType)
// 	case "object":
// 		return marshalBody(body, op.Response.BodyMediaType)
// 	}
// 	return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
// }

func (op *OperationStore) Parameterize(parentDoc *Service, inputParams map[string]interface{}, requestBody interface{}) (*openapi3filter.RequestValidationInput, error) {
	params := op.OperationRef.Value.Parameters
	copyParams := make(map[string]interface{})
	for k, v := range inputParams {
		copyParams[k] = v
	}
	pathParams := make(map[string]string)
	q := make(url.Values)
	for _, p := range params {
		if p.Value == nil {
			continue
		}
		name := p.Value.Name
		if p.Value.In == openapi3.ParameterInPath {
			val, present := copyParams[p.Value.Name]
			if present {
				pathParams[name] = fmt.Sprintf("%v", val)
				delete(copyParams, name)
			}
			if !present && p.Value.Required {
				return nil, fmt.Errorf("OperationStore.Parameterize() failure")
			}
		} else if p.Value.In == openapi3.ParameterInQuery {
			val, present := copyParams[p.Value.Name]
			if present {
				q.Set(name, fmt.Sprintf("%v", val))
				delete(copyParams, name)
			}
		}
	}
	router, err := queryrouter.NewRouter(parentDoc.GetT())
	if err != nil {
		return nil, err
	}
	servers := getServersFromHeirarchy(parentDoc, op)
	sv, err := selectServer(servers, inputParams)
	if err != nil {
		return nil, err
	}
	contentTypeHeaderRequired := false
	var bodyReader io.Reader
	if requestBody != nil && op.Request != nil {
		b, err := marshalBody(requestBody, op.Request.BodyMediaType)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
		contentTypeHeaderRequired = true
	}
	// TODO: clean up
	sv = strings.TrimSuffix(sv, "/")
	path := replaceSimpleStringVars(fmt.Sprintf("%s%s", sv, op.OperationRef.extractPathItem()), pathParams)
	u, err := url.Parse(fmt.Sprintf("%s?%s", path, q.Encode()))
	if strings.Contains(path, "?") {
		if len(q) > 0 {
			u, err = url.Parse(fmt.Sprintf("%s&%s", path, q.Encode()))
		} else {
			u, err = url.Parse(path)
		}
	}
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(strings.ToUpper(op.OperationRef.extractMethodItem()), u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	if contentTypeHeaderRequired {
		httpReq.Header.Set("Content-Type", op.Request.BodyMediaType)
	}
	route, checkedPathParams, err := router.FindRoute(httpReq)
	if err != nil {
		return nil, err
	}
	options := &openapi3filter.Options{
		AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
	}
	// Validate request
	requestValidationInput := &openapi3filter.RequestValidationInput{
		Options:    options,
		PathParams: checkedPathParams,
		Request:    httpReq,
		Route:      route,
	}
	return requestValidationInput, nil
}

func (op *OperationStore) GetRequestBodySchema() (*Schema, error) {
	if op.Request != nil {
		return op.Request.Schema, nil
	}
	return nil, fmt.Errorf("no request body for operation =  %s", op.GetName())
}

func (op *OperationStore) GetRequestBodyRequiredProperties() ([]string, error) {
	if op.Request != nil {
		return op.Request.Required, nil
	}
	return nil, fmt.Errorf("no request body required elements for operation =  %s", op.GetName())
}

func (op *OperationStore) IsRequiredRequestBodyProperty(key string) bool {
	if op.Request == nil || op.Request.Required == nil {
		return false
	}
	for _, k := range op.Request.Required {
		if k == key {
			return true
		}
	}
	return false
}

func (op *OperationStore) GetResponseBodySchemaAndMediaType() (*Schema, string, error) {
	return op.getResponseBodySchemaAndMediaType()
}

func (op *OperationStore) getResponseBodySchemaAndMediaType() (*Schema, string, error) {
	if op.Response != nil && op.Response.Schema != nil {
		return op.Response.Schema, op.Response.BodyMediaType, nil
	}
	return nil, "", fmt.Errorf("no response body for operation =  %s", op.GetName())
}

func (op *OperationStore) GetSelectSchemaAndObjectPath() (*Schema, string, error) {
	k := op.lookupSelectItemsKey()
	if op.Response != nil && op.Response.Schema != nil {
		return op.Response.Schema.getSelectItemsSchema(k, op.getOptimalResponseMediaType())
	}
	return nil, "", fmt.Errorf("no response body for operation =  %s", op.GetName())
}

func (op *OperationStore) ProcessResponse(response *http.Response) (interface{}, error) {
	responseSchema, _, err := op.GetResponseBodySchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	return responseSchema.ProcessHttpResponse(response, op.lookupSelectItemsKey())
}

func (ops *OperationStore) lookupSelectItemsKey() string {
	s := ops.getSelectItemsKeySimple()
	if s != "" {
		return s
	}
	responseSchema, _, err := ops.GetResponseBodySchemaAndMediaType()
	if responseSchema == nil || err != nil {
		return ""
	}
	switch responseSchema.Type {
	case "string", "integer":
		return AnonymousColumnName
	}
	if _, ok := responseSchema.getProperty(defaultSelectItemsKey); ok {
		return defaultSelectItemsKey
	}
	return ""
}

func (op *OperationStore) DeprecatedProcessResponse(response *http.Response) (map[string]interface{}, error) {
	responseSchema, _, err := op.GetResponseBodySchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	return responseSchema.DeprecatedProcessHttpResponse(response, op.lookupSelectItemsKey())
}
