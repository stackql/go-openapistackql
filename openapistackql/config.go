package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

var (
	_ jsonpointer.JSONPointable = standardStackQLConfig{}
	_ StackQLConfig             = &standardStackQLConfig{}
)

type StackQLConfig interface {
	GetAuth() (AuthDTO, bool)
	GetViewBodyDDLForSQLDialect(sqlDialect string, viewName string) (string, bool)
	GetQueryTranspose() Transform
	GetRequestTranslate() Transform
	GetPagination() Pagination
	GetVariations() Variations
	GetViews() map[string]View
	GetExternalTables() map[string]SQLExternalTable
	//
	isObjectSchemaImplicitlyUnioned() bool
}

type standardStackQLConfig struct {
	QueryTranspose   *standardTransform                  `json:"queryParamTranspose,omitempty" yaml:"queryParamTranspose,omitempty"`
	RequestTranslate *standardTransform                  `json:"requestTranslate,omitempty" yaml:"requestTranslate,omitempty"`
	Pagination       *standardPagination                 `json:"pagination,omitempty" yaml:"pagination,omitempty"`
	Variations       *standardVariations                 `json:"variations,omitempty" yaml:"variations,omitempty"`
	Views            map[string]*standardView            `json:"views" yaml:"views"`
	ExternalTables   map[string]standardSQLExternalTable `json:"sqlExternalTables" yaml:"sqlExternalTables"`
	Auth             *standardAuthDTO                    `json:"auth,omitempty" yaml:"auth,omitempty"`
}

func (qt standardStackQLConfig) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "queryTranspose":
		return qt.QueryTranspose, nil
	case "views":
		return qt.Views, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from QueryTranspose doc object", token)
	}
}

func (cfg *standardStackQLConfig) GetQueryTranspose() Transform {
	return cfg.QueryTranspose
}

func (cfg *standardStackQLConfig) GetRequestTranslate() Transform {
	return cfg.RequestTranslate
}

func (cfg *standardStackQLConfig) GetPagination() Pagination {
	return cfg.Pagination
}

func (cfg *standardStackQLConfig) GetVariations() Variations {
	return cfg.Variations
}

func (cfg *standardStackQLConfig) GetViews() map[string]View {
	rv := make(map[string]View, len(cfg.Views))
	if cfg.Views != nil {
		for k, v := range cfg.Views {
			rv[k] = v
		}
	}
	return rv
}

func (cfg *standardStackQLConfig) isObjectSchemaImplicitlyUnioned() bool {
	if cfg.Variations != nil {
		return cfg.Variations.IsObjectSchemaImplicitlyUnioned()
	}
	return false
}

func (cfg *standardStackQLConfig) GetView(viewName string) (View, bool) {
	if cfg.Views != nil {
		v, ok := cfg.Views[viewName]
		return v, ok
	}
	return nil, false
}

func (cfg *standardStackQLConfig) GetAuth() (AuthDTO, bool) {
	return cfg.Auth, cfg.Auth != nil
}

func (cfg *standardStackQLConfig) GetExternalTables() map[string]SQLExternalTable {
	rv := make(map[string]SQLExternalTable, len(cfg.ExternalTables))
	if cfg.ExternalTables != nil {
		for k, v := range cfg.ExternalTables {
			rv[k] = v
		}
		return rv
	}
	return nil
}

func (cfg *standardStackQLConfig) GetViewBodyDDLForSQLDialect(sqlDialect string, viewName string) (string, bool) {
	if cfg.Views != nil {
		v, ok := cfg.Views[viewName]
		if !ok || v == nil {
			return "", false
		}
		return v.GetDDLForSqlDialect(sqlDialect)
	}
	return "", false
}
