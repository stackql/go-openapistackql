package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type StackQLConfig struct {
	QueryTranspose *QueryTranspose `json:"queryParamTranspose,omitempty" yaml:"queryParamTranspose,omitempty"`
	Pagination     *Pagination     `json:"pagination,omitempty" yaml:"pagination,omitempty"`
}

var _ jsonpointer.JSONPointable = (StackQLConfig)(StackQLConfig{})

func (qt StackQLConfig) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "queryTranspose":
		return qt.QueryTranspose, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from QueryTranspose doc object", token)
	}
}
