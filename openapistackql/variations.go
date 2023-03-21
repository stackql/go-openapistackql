package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

var (
	_ Variations                = standardVariations{}
	_ jsonpointer.JSONPointable = (Variations)(standardVariations{})
)

type Variations interface {
	JSONLookup(token string) (interface{}, error)
	IsObjectSchemaImplicitlyUnioned() bool
}

type standardVariations struct {
	IsObjectSchemaImplicitlyUnionedVal bool `json:"isObjectSchemaImplicitlyUnioned,omitempty" yaml:"isObjectSchemaImplicitlyUnioned,omitempty"`
}

func (qt standardVariations) IsObjectSchemaImplicitlyUnioned() bool {
	return qt.IsObjectSchemaImplicitlyUnionedVal
}

func (qt standardVariations) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "isObjectImplicitlyUnioned":
		return qt.IsObjectSchemaImplicitlyUnioned, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from QueryTranspose doc object", token)
	}
}
