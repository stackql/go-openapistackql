package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type GraphQLCursor map[string]interface{}

func (gqc GraphQLCursor) GetCursorJSONPath() (string, bool) {
	jp, ok := gqc["jsonPath"]
	switch jp := jp.(type) {
	case string:
		return jp, ok
	default:
		return "", false
	}
}

type GraphQL struct {
	ID       string        `json:"id" yaml:"id"`
	Query    string        `json:"query,omitempty" yaml:"query,omitempty"` // Required
	Cursor   GraphQLCursor `json:"cursor,omitempty" yaml:"cursor,omitempty"`
	URL      string        `json:"url" yaml:"url"`
	HTTPVerb string        `json:"httpVerb" yaml:"httpVerb"`
}

var _ jsonpointer.JSONPointable = (GraphQL)(GraphQL{})

func (gq GraphQL) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "id":
		return gq.ID, nil
	case "query":
		return gq.Query, nil
	case "cursor":
		return gq.Cursor, nil
	case "url":
		return gq.URL, nil
	case "httpVerb":
		return gq.HTTPVerb, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from GraphQL doc object", token)
	}
}
