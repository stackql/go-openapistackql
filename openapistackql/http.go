package openapistackql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackql/go-openapistackql/pkg/querytranspose"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	ParamEncodeDelimiter string = "%"
)

type ParameterBinding struct {
	Param Addressable // may originally be *openapi3.Parameter or *openapi3.ServerVariable, latter will be co-opted
	Val   interface{}
}

type ParamMap map[string]ParameterBinding

type ParamPair struct {
	Key   string
	Param ParameterBinding
}

type BodyMap map[string]interface{}

type BodyParamPair struct {
	Key string
	Val interface{}
}

type EncodableString string

func (es EncodableString) encodeWithPrefixAndKey(prefix, key string) string {
	return ParamEncodeDelimiter + prefix + ParamEncodeDelimiter + key + ParamEncodeDelimiter + string(es) + ParamEncodeDelimiter
}

func (pm BodyMap) order() []BodyParamPair {
	var rv []BodyParamPair
	for k, v := range pm {
		rv = append(rv, BodyParamPair{Key: k, Val: v})
	}
	sort.Slice(rv, func(i, j int) bool {
		return rv[i].Key < rv[j].Key
	})
	return rv
}

func (pm ParamMap) order() []ParamPair {
	var rv []ParamPair
	for k, v := range pm {
		rv = append(rv, ParamPair{Key: k, Param: v})
	}
	sort.Slice(rv, func(i, j int) bool {
		return rv[i].Key < rv[j].Key
	})
	return rv
}

func (bm BodyMap) encodeWithPrefix(prefix string) string {
	var sb strings.Builder
	for _, v := range bm.order() {
		sb.WriteString(ParamEncodeDelimiter + prefix + ParamEncodeDelimiter + v.Key + ParamEncodeDelimiter + fmt.Sprintf("%v", v.Val) + ParamEncodeDelimiter)
	}
	return sb.String()
}

func (pm ParamMap) encodeWithPrefix(prefix string) string {
	var sb strings.Builder
	for _, v := range pm.order() {
		sb.WriteString(ParamEncodeDelimiter + prefix + ParamEncodeDelimiter + v.Key + ParamEncodeDelimiter + fmt.Sprintf("%v", v.Param.Val) + ParamEncodeDelimiter)
	}
	return sb.String()
}

func NewParameterBinding(param Addressable, val interface{}) ParameterBinding {
	return ParameterBinding{
		Param: param,
		Val:   val,
	}
}

type HttpParameters struct {
	opStore      *OperationStore
	CookieParams ParamMap
	HeaderParams ParamMap
	PathParams   ParamMap
	QueryParams  ParamMap
	RequestBody  BodyMap
	ResponseBody BodyMap
	ServerParams ParamMap
	Unassigned   ParamMap
	Region       EncodableString
}

func NewHttpParameters(method *OperationStore) *HttpParameters {
	return &HttpParameters{
		opStore:      method,
		CookieParams: make(ParamMap),
		HeaderParams: make(ParamMap),
		PathParams:   make(ParamMap),
		QueryParams:  make(ParamMap),
		RequestBody:  make(BodyMap),
		ResponseBody: make(BodyMap),
		ServerParams: make(ParamMap),
		Unassigned:   make(ParamMap),
	}
}

func (hp *HttpParameters) Encode() string {
	var sb strings.Builder
	sb.WriteString(hp.CookieParams.encodeWithPrefix("cookie"))
	sb.WriteString(hp.HeaderParams.encodeWithPrefix("header"))
	sb.WriteString(hp.PathParams.encodeWithPrefix("path"))
	sb.WriteString(hp.QueryParams.encodeWithPrefix("query"))
	sb.WriteString(hp.RequestBody.encodeWithPrefix("requestBody"))
	sb.WriteString(hp.Region.encodeWithPrefixAndKey("region", "region"))
	sb.WriteString(hp.ServerParams.encodeWithPrefix("server"))
	return sb.String()
}

func (hp *HttpParameters) IngestMap(m map[string]interface{}) error {
	for k, v := range m {
		if param, ok := hp.opStore.GetOperationParameter(k); ok {
			hp.StoreParameter(param, v)
		} else if _, ok := hp.opStore.getServerVariable(k); ok {
			param := &openapi3.Parameter{
				In:   "server",
				Name: k,
			}
			svc := hp.opStore.Service
			hp.StoreParameter(NewParameter(param, svc), v)
		} else {
			return fmt.Errorf("could not place parameter '%s'", k)
		}
	}
	return nil
}

func (hp *HttpParameters) StoreParameter(param Addressable, val interface{}) {
	if param.GetLocation() == openapi3.ParameterInPath {
		hp.PathParams[param.GetName()] = NewParameterBinding(param, val)
		return
	}
	if param.GetLocation() == openapi3.ParameterInQuery {
		hp.QueryParams[param.GetName()] = NewParameterBinding(param, val)
		return
	}
	if param.GetLocation() == openapi3.ParameterInHeader {
		hp.HeaderParams[param.GetName()] = NewParameterBinding(param, val)
		return
	}
	if param.GetLocation() == openapi3.ParameterInCookie {
		hp.CookieParams[param.GetName()] = NewParameterBinding(param, val)
		return
	}
	if param.GetLocation() == "server" {
		hp.ServerParams[param.GetName()] = NewParameterBinding(param, val)
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
			_, ok := keysRemaining[mk]
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
