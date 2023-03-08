package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type StackQLConfig struct {
	QueryTranspose   *Transform                  `json:"queryParamTranspose,omitempty" yaml:"queryParamTranspose,omitempty"`
	RequestTranslate *Transform                  `json:"requestTranslate,omitempty" yaml:"requestTranslate,omitempty"`
	Pagination       *Pagination                 `json:"pagination,omitempty" yaml:"pagination,omitempty"`
	Variations       *Variations                 `json:"variations,omitempty" yaml:"variations,omitempty"`
	Views            map[string]*View            `json:"views" yaml:"views"`
	ExternalTables   map[string]SQLExternalTable `json:"sqlExternalTables" yaml:"sqlExternalTables"`
	Auth             *AuthDTO                    `json:"auth,omitempty" yaml:"auth,omitempty"`
}

var _ jsonpointer.JSONPointable = (StackQLConfig)(StackQLConfig{})

func (qt StackQLConfig) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "queryTranspose":
		return qt.QueryTranspose, nil
	case "views":
		return qt.Views, nil
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
		return v, ok
	}
	return nil, false
}

func (cfg *StackQLConfig) GetAuth() (*AuthDTO, bool) {
	return cfg.Auth, cfg.Auth != nil
}

func (cfg *StackQLConfig) GetViewBodyDDLForSQLDialect(sqlDialect string, viewName string) (string, bool) {
	if cfg.Views != nil {
		v, ok := cfg.Views[viewName]
		if !ok || v == nil {
			return "", false
		}
		return v.GetDDLForSqlDialect(sqlDialect)
	}
	return "", false
}

func (cfg *StackQLConfig) GetViews(viewName string) (*View, bool) {
	if cfg.Views != nil {
		v, ok := cfg.Views[viewName]
		return v, ok
	}
	return nil, false
}
