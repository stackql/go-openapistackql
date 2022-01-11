package openapistackql

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type Parameter openapi3.Parameter

type Parameters openapi3.Parameters

func (p *Parameter) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(p.Schema.Value.Type, lhs, rhs)
}

func (p *Parameter) GetType() string {
	return p.Schema.Value.Type
}

func (p Parameters) getParameterFromInSubset(key, inSubset string) (*Parameter, bool) {
	for _, paramRef := range p {
		param := paramRef.Value
		if param.In == inSubset && param.Name == key {
			return (*Parameter)(param), true
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
