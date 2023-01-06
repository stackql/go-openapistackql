package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type StackQLConfig struct {
	QueryTranspose   *Transform      `json:"queryParamTranspose,omitempty" yaml:"queryParamTranspose,omitempty"`
	RequestTranslate *Transform      `json:"requestTranslate,omitempty" yaml:"requestTranslate,omitempty"`
	Pagination       *Pagination     `json:"pagination,omitempty" yaml:"pagination,omitempty"`
	Variations       *Variations     `json:"variations,omitempty" yaml:"variations,omitempty"`
	Views            map[string]View `json:"views" yaml:"views"`
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

func (cfg *StackQLConfig) isObjectSchemaImplicitlyUnioned() bool {
	if cfg.Variations != nil {
		return cfg.Variations.IsObjectSchemaImplicitlyUnioned
	}
	return false
}

func (cfg *StackQLConfig) GetView(viewName string) (*View, bool) {
	if cfg.Views != nil {
		v, ok := cfg.Views[viewName]
		return &v, ok
	}
	return nil, false
}
