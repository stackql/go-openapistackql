package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type TokenSemanticArgs map[string]interface{}

type TokenSemantic struct {
	Algorithm string            `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	Args      TokenSemanticArgs `json:"args,omitempty" yaml:"args,omitempty"`
	Key       string            `json:"key,omitempty" yaml:"key,omitempty"`
	Location  string            `json:"location,omitempty" yaml:"location,omitempty"`
}

func (ts *TokenSemantic) GetTransformer() (TokenTransformer, error) {
	tl := NewStandardTransformerLocator()
	return tl.GetTransformer(ts)
}

var _ jsonpointer.JSONPointable = (TokenSemantic)(TokenSemantic{})

func (qt TokenSemantic) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "algorithm":
		return qt.Algorithm, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from TokenSemantic doc object", token)
	}
}

func (tsa TokenSemanticArgs) GetRegex() (string, bool) {
	r, ok := tsa["regex"]
	if !ok {
		return "", false
	}
	rv, ok := r.(string)
	if !ok {
		return "", false
	}
	return rv, true
}
