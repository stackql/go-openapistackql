package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type View struct {
	SQLBackend  string `json:"sqlBackend" yaml:"sqlBackend"`
	DDL         string `json:"ddl" yaml:"ddl"`
	FallbackDDL string `json:"fallbackDdl" yaml:"fallbackDdl"` // Future proofing for predicate failover
}

var _ jsonpointer.JSONPointable = (View)(View{})

func (qt View) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "ddl":
		return qt.DDL, nil
	case "sqlBackend":
		return qt.SQLBackend, nil
	case "fallbackDdl":
		return qt.FallbackDDL, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from View doc object", token)
	}
}
