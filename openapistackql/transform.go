package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type Transform interface {
	JSONLookup(token string) (interface{}, error)
	GetAlgorithm() string
}

type standardTransform struct {
	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
}

func (ts standardTransform) GetAlgorithm() string {
	return ts.Algorithm
}

var _ jsonpointer.JSONPointable = (Transform)(standardTransform{})

func (qt standardTransform) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "algorithm":
		return qt.Algorithm, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from Transform doc object", token)
	}
}
