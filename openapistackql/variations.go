package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type Variations struct {
	IsObjectSchemaImplicitlyUnioned bool `json:"isObjectSchemaImplicitlyUnioned,omitempty" yaml:"isObjectSchemaImplicitlyUnioned,omitempty"`
}

var _ jsonpointer.JSONPointable = (Variations)(Variations{})

func (qt Variations) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "isObjectImplicitlyUnioned":
		return qt.IsObjectSchemaImplicitlyUnioned, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from QueryTranspose doc object", token)
	}
}
