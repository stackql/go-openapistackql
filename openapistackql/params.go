package openapistackql

import (
	"github.com/getkin/kin-openapi/openapi3"
)

var (
	// Enforce invariant
	_ Addressable = &standardParameter{}
	_ Params      = &parameters{}
)

type standardParameter struct {
	openapi3.Parameter
	svc Service
}

func NewParameter(param *openapi3.Parameter, svc Service) Addressable {
	return &standardParameter{
		*param,
		svc,
	}
}

type Params interface {
	GetParameter(key string) (Addressable, bool)
}

type parameters struct {
	openapi3.Parameters
	svc Service
}

func NewParameters(params openapi3.Parameters, svc Service) Params {
	return parameters{
		params,
		svc,
	}
}

func (p *standardParameter) GetName() string {
	return p.Name
}

func (p *standardParameter) GetLocation() string {
	return p.In
}

func (p *standardParameter) GetSchema() (Schema, bool) {
	if p.Schema != nil && p.Schema.Value != nil {
		return NewSchema(p.Schema.Value, p.svc, "", p.Schema.Ref), true
	}
	return nil, false
}

func (p *standardParameter) IsRequired() bool {
	return isOpenapi3ParamRequired(&p.Parameter)
}

func isOpenapi3ParamRequired(param *openapi3.Parameter) bool {
	return param.Required && !param.AllowEmptyValue
}

func (p *standardParameter) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(p.Schema.Value.Type, lhs, rhs)
}

func (p *standardParameter) GetType() string {
	return p.Schema.Value.Type
}

func (p parameters) getParameterFromInSubset(key, inSubset string) (Addressable, bool) {
	for _, paramRef := range p.Parameters {
		param := paramRef.Value
		if param.In == inSubset && param.Name == key {
			return NewParameter(param, p.svc), true
		}
	}
	return nil, false
}

func (p parameters) GetParameter(key string) (Addressable, bool) {
	if param, ok := p.getParameterFromInSubset(key, openapi3.ParameterInPath); ok {
		return param, true
	}
	if param, ok := p.getParameterFromInSubset(key, openapi3.ParameterInQuery); ok {
		return param, true
	}
	if param, ok := p.getParameterFromInSubset(key, openapi3.ParameterInHeader); ok {
		return param, true
	}
	if param, ok := p.getParameterFromInSubset(key, openapi3.ParameterInCookie); ok {
		return param, true
	}
	return nil, false
}
