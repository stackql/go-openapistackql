package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type Transform struct {
	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
}

var _ jsonpointer.JSONPointable = (Transform)(Transform{})

func (qt Transform) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "algorithm":
		return qt.Algorithm, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from Transform doc object", token)
	}
}
