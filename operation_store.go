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

	openapirouter "github.com/getkin/kin-openapi/routers/gorillamux"

	"vitess.io/vitess/go/sqltypes"
)

type Methods map[string]OperationStore

func (ms Methods) FindMethod(key string) (*OperationStore, error) {
	if m, ok := ms[key]; ok {
		return &m, nil
	}
	return nil, fmt.Errorf("could not find method for key = '%s'", key)
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
	SQLVerb   string `json:"sqlVerb" yaml:"sqlVerb"` // Required
	// Optional parameters.
	Parameters   map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	PathItemRef  *PathItemRef           `json:"path" yaml:"path"`           // Required
	APIMethod    string                 `json:"apiMethod" yaml:"apiMethod"` // Required
	OperationRef *OperationRef          `json:"operation" yaml:"operation"` // Required
	Request      *ExpectedRequest       `json:"request" yaml:"request"`
	Response     *ExpectedResponse      `json:"response" yaml:"response"`
	Servers      *openapi3.Servers      `json:"servers" yaml:"servers"`
	// private
	parameterizedPath string `json:"-" yaml:"-"`
}

func (op *OperationStore) GetParameterizedPath() string {
	return op.parameterizedPath
}

func (op *OperationStore) IsNullary() bool {
	rbs, _ := op.GetResponseBodySchema()
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
	}
	if extended {
		retVal = append(retVal, MethodDescription)
	}
	return retVal
}

func (m *OperationStore) IsAwaitable() bool {
	rs, err := m.GetResponseBodySchema()
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
	for _, s := range requiredBodyParamNames {
		requiredParamNames = append(requiredParamNames, s)
	}
	retVal := map[string]interface{}{
		MethodName:     m.MethodKey,
		RequiredParams: strings.Join(requiredParamNames, ", "),
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

func (op *OperationStore) ProcessResponse(body []byte) (interface{}, error) {
	switch op.Response.Schema.Type {
	case "string": // (this includes dates and files)
		return string(body), nil
	case "number":
		return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
	case "integer":
		return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
	case "boolean":
		return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
	case "array":
		return marshalBody(body, op.Response.BodyMediaType)
	case "object":
		return marshalBody(body, op.Response.BodyMediaType)
	}
	return nil, fmt.Errorf("raw %T as top-level response not currently supported", op.Response.Schema.Type)
}

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
	router, err := openapirouter.NewRouter(parentDoc.GetT())
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
	path := replaceSimpleStringVars(fmt.Sprintf("%s%s", sv, op.PathItemRef.Ref), pathParams)
	u, err := url.Parse(fmt.Sprintf("%s?%s", path, q.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(strings.ToUpper(op.OperationRef.Ref), u.String(), bodyReader)
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

func (op *OperationStore) GetResponseBodySchema() (*Schema, error) {
	if op.Response != nil {
		return op.Response.Schema, nil
	}
	return nil, fmt.Errorf("no response body for operation =  %s", op.GetName())
}
