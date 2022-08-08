package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type QueryTranspose struct {
	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
}

var _ jsonpointer.JSONPointable = (QueryTranspose)(QueryTranspose{})

func (qt QueryTranspose) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "algorithm":
		return qt.Algorithm, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from QueryTranspose doc object", token)
	}
}
