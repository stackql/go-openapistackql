package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

var (
	_ OperationInverse          = &operationInverse{}
	_ jsonpointer.JSONPointable = (OperationInverse)(&operationInverse{})
	_ OperationTokens           = operationTokens{}
	_ jsonpointer.JSONPointable = operationTokens{}
)

type OperationTokens interface {
	JSONLookup(token string) (interface{}, error)
	GetTokenSemantic(key string) (TokenSemantic, bool)
}

type operationTokens map[string]standardTokenSemantic

func (oits operationTokens) JSONLookup(token string) (interface{}, error) {
	if tokenSemantic, ok := oits[token]; ok {
		return tokenSemantic, nil
	}
	return nil, fmt.Errorf("could not resolve token '%s' from OperationInverseTokens doc object", token)
}

func (oits operationTokens) GetTokenSemantic(key string) (TokenSemantic, bool) {
	tokenSemantic, ok := oits[key]
	return &tokenSemantic, ok
}

type OperationInverse interface {
	JSONLookup(token string) (interface{}, error)
	GetOperationStore() (OperationStore, bool)
	GetTokens() (OperationTokens, bool)
}

type operationInverse struct {
	OpRef         OperationStoreRef `json:"operation" yaml:"operation"`
	ReverseTokens operationTokens   `json:"tokens,omitempty" yaml:"tokens,omitempty"`
}

func (oi *operationInverse) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "inverse":
		return oi.OpRef, nil
	case "tokens":
		return oi.ReverseTokens, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from OperationInverse doc object", token)
	}
}

func (oi *operationInverse) GetOperationStore() (OperationStore, bool) {
	if oi.OpRef.Ref == "" || oi.OpRef.Value == nil {
		return nil, false
	}
	return oi.OpRef.Value, true
}

func (oi *operationInverse) GetTokens() (OperationTokens, bool) {
	return oi.ReverseTokens, oi.ReverseTokens != nil
}
