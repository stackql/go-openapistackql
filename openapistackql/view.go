package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type View struct {
	Predicate string `json:"predicate" yaml:"predicate"`
	DDL       string `json:"ddl" yaml:"ddl"`
	Fallback  *View  `json:"fallback" yaml:"fallback"` // Future proofing for predicate failover
}

var _ jsonpointer.JSONPointable = (View)(View{})

func (qt View) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "ddl":
		return qt.DDL, nil
	case "predicate":
		return qt.Predicate, nil
	case "fallback":
		return qt.Fallback, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from View doc object", token)
	}
}
