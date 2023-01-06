package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type View struct {
	DDL string `json:"ddl" yaml:"ddl"`
}

var _ jsonpointer.JSONPointable = (View)(View{})

func (qt View) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "ddl":
		return qt.DDL, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from View doc object", token)
	}
}
