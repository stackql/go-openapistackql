package openapistackql

import (
	"fmt"
	"strings"

	"github.com/stackql/go-openapistackql/pkg/querytranspose"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/getkin/kin-openapi/openapi3"
)

type ParameterBinding struct {
	Param *Parameter // may originally be *openapi3.Parameter or *openapi3.ServerVariable, latter will be co-opted
	Val   interface{}
}

func NewParameterBinding(param *Parameter, val interface{}) ParameterBinding {
	return ParameterBinding{
		Param: param,
		Val:   val,
	}
}

type HttpParameters struct {
	opStore      *OperationStore
	CookieParams map[string]ParameterBinding
	HeaderParams map[string]ParameterBinding
	PathParams   map[string]ParameterBinding
	QueryParams  map[string]ParameterBinding
	RequestBody  map[string]interface{}
	ResponseBody map[string]interface{}
	ServerParams map[string]ParameterBinding
	Unassigned   map[string]ParameterBinding
	Region       string
}

func NewHttpParameters(method *OperationStore) *HttpParameters {
	return &HttpParameters{
		opStore:      method,
		CookieParams: make(map[string]ParameterBinding),
		HeaderParams: make(map[string]ParameterBinding),
		PathParams:   make(map[string]ParameterBinding),
		QueryParams:  make(map[string]ParameterBinding),
		RequestBody:  make(map[string]interface{}),
		ResponseBody: make(map[string]interface{}),
		ServerParams: make(map[string]ParameterBinding),
		Unassigned:   make(map[string]ParameterBinding),
	}
}

func (hp *HttpParameters) IngestMap(m map[string]interface{}) error {
	for k, v := range m {
		if param, ok := hp.opStore.GetOperationParameter(k); ok {
			hp.StoreParameter(param, v)
		} else if _, ok := hp.opStore.getServerVariable(k); ok {
			param := &Parameter{
				In:   "server",
				Name: k,
			}
			hp.StoreParameter(param, v)
		} else {
			return fmt.Errorf("could not place parameter '%s'", k)
		}
	}
	return nil
}

func (hp *HttpParameters) StoreParameter(param *Parameter, val interface{}) {
	if param.In == openapi3.ParameterInPath {
		hp.PathParams[param.Name] = NewParameterBinding(param, val)
		return
	}
	if param.In == openapi3.ParameterInQuery {
		hp.QueryParams[param.Name] = NewParameterBinding(param, val)
		return
	}
	if param.In == openapi3.ParameterInHeader {
		hp.HeaderParams[param.Name] = NewParameterBinding(param, val)
		return
	}
	if param.In == openapi3.ParameterInCookie {
		hp.CookieParams[param.Name] = NewParameterBinding(param, val)
		return
	}
	if param.In == "server" {
		hp.ServerParams[param.Name] = NewParameterBinding(param, val)
		return
	}
}

func (hp *HttpParameters) GetParameter(paramName, paramIn string) (*ParameterBinding, bool) {
	if paramIn == openapi3.ParameterInPath {
		rv, ok := hp.PathParams[paramName]
		if !ok {
			return nil, false
		}
		return &rv, true
	}
	if paramIn == openapi3.ParameterInQuery {
		rv, ok := hp.QueryParams[paramName]
		if !ok {
			return nil, false
		}
		return &rv, true
	}
	if paramIn == openapi3.ParameterInHeader {
		rv, ok := hp.HeaderParams[paramName]
		if !ok {
			return nil, false
		}
		return &rv, true
	}
	if paramIn == openapi3.ParameterInCookie {
		rv, ok := hp.CookieParams[paramName]
		if !ok {
			return nil, false
		}
		return &rv, true
	}
	if paramIn == "server" {
		rv, ok := hp.CookieParams[paramName]
		if !ok {
			return nil, false
		}
		return &rv, true
	}
	return nil, false
}

func (hp *HttpParameters) processFuncHTTPParam(key string, param interface{}) (map[string]string, error) {
	switch param := param.(type) {
	case *sqlparser.FuncExpr:
		if strings.ToUpper(param.Name.GetRawVal()) == "JSON" {
			if len(param.Exprs) != 1 {
				return nil, fmt.Errorf("cannot accomodate JSON Function with arg count = %d", len(param.Exprs))
			}
			switch ex := param.Exprs[0].(type) {
			case *sqlparser.AliasedExpr:
				switch argExpr := ex.Expr.(type) {
				case *sqlparser.SQLVal:
					queryTransposer := querytranspose.NewQueryTransposer(hp.opStore.GetQueryTransposeAlgorithm(), argExpr.Val, key)
					return queryTransposer.Transpose()
				default:
					return nil, fmt.Errorf("cannot process json function underlying arg of type = '%T'", argExpr)
				}
			default:
				return nil, fmt.Errorf("cannot process json function arg of type = '%T'", ex)
			}
		}
	}
	return map[string]string{key: fmt.Sprintf("%v", param)}, nil
}

func (hp *HttpParameters) updateStuff(k string, v ParameterBinding, paramMap map[string]interface{}, visited map[string]struct{}) error {
	if _, ok := visited[k]; ok {
		return fmt.Errorf("parameter name = '%s' repeated, cannot convert to flat map", k)
	}
	paramMap[k] = v.Val
	visited[k] = struct{}{}
	return nil
}

func (hp *HttpParameters) ToFlatMap() (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	visited := make(map[string]struct{})
	for k, v := range hp.CookieParams {
		err := hp.updateStuff(k, v, rv, visited)
		if err != nil {
			return nil, err
		}
	}
	for k, v := range hp.HeaderParams {
		err := hp.updateStuff(k, v, rv, visited)
		if err != nil {
			return nil, err
		}
	}
	for k, v := range hp.PathParams {
		err := hp.updateStuff(k, v, rv, visited)
		if err != nil {
			return nil, err
		}
	}
	for k, v := range hp.QueryParams {
		// var err error
		m, err := hp.processFuncHTTPParam(k, v.Val)
		if err != nil {
			return nil, err
		}
		for mk, mv := range m {
			val := NewParameterBinding(nil, mv)
			err = hp.updateStuff(mk, val, rv, visited)
			if err != nil {
				return nil, err
			}
		}
	}
	for k, v := range hp.ServerParams {
		err := hp.updateStuff(k, v, rv, visited)
		if err != nil {
			return nil, err
		}
	}
	return rv, nil
}

func (hp *HttpParameters) GetServerParameterFlatMap() (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	visited := make(map[string]struct{})
	for k, v := range hp.ServerParams {
		err := hp.updateStuff(k, v, rv, visited)
		if err != nil {
			return nil, err
		}
	}
	return rv, nil
}

func (hp *HttpParameters) GetRemainingQueryParamsFlatMap(keysRemaining map[string]interface{}) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	visited := make(map[string]struct{})
	for k, v := range hp.QueryParams {
		// var err error
		m, err := hp.processFuncHTTPParam(k, v.Val)
		if err != nil {
			return nil, err
		}
		for mk, mv := range m {
			_, ok := keysRemaining[k]
			if !ok {
				continue
			}
			val := NewParameterBinding(nil, mv)
			err = hp.updateStuff(mk, val, rv, visited)
			if err != nil {
				return nil, err
			}
		}
	}
	return rv, nil
}
