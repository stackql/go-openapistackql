package openapistackql

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type Parameter struct {
	openapi3.Parameter
	svc *Service
}

func NewParameter(param *openapi3.Parameter, svc *Service) *Parameter {
	return &Parameter{
		*param,
		svc,
	}
}

// Enforce invariant
var _ Addressable = &Parameter{}

type Parameters struct {
	openapi3.Parameters
	svc *Service
}

func NewParameters(params openapi3.Parameters, svc *Service) Parameters {
	return Parameters{
		params,
		svc,
	}
}

func (p *Parameter) GetName() string {
	return p.Name
}

func (p *Parameter) GetLocation() string {
	return p.In
}

func (p *Parameter) GetSchema() (Schema, bool) {
	if p.Schema != nil && p.Schema.Value != nil {
		return NewSchema(p.Schema.Value, p.svc, "", p.Schema.Ref), true
	}
	return nil, false
}

func (p *Parameter) IsRequired() bool {
	return isOpenapi3ParamRequired(&p.Parameter)
}

func isOpenapi3ParamRequired(param *openapi3.Parameter) bool {
	return param.Required && !param.AllowEmptyValue
}

func (p *Parameter) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(p.Schema.Value.Type, lhs, rhs)
}

func (p *Parameter) GetType() string {
	return p.Schema.Value.Type
}

func (p Parameters) getParameterFromInSubset(key, inSubset string) (*Parameter, bool) {
	for _, paramRef := range p.Parameters {
		param := paramRef.Value
		if param.In == inSubset && param.Name == key {
			return NewParameter(param, p.svc), true
		}
	}
	return nil, false
}

func (p Parameters) GetParameter(key string) (*Parameter, bool) {
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
