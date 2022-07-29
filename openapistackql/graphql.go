package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type GraphQLElement map[string]interface{}

func (gqc GraphQLElement) getJSONPath() (string, bool) {
	jp, ok := gqc["jsonPath"]
	switch jp := jp.(type) {
	case string:
		return jp, ok
	default:
		return "", false
	}
}

type GraphQL struct {
	ID               string         `json:"id" yaml:"id"`
	Query            string         `json:"query,omitempty" yaml:"query,omitempty"` // Required
	Cursor           GraphQLElement `json:"cursor,omitempty" yaml:"cursor,omitempty"`
	ReponseSelection GraphQLElement `json:"responseSelection,omitempty" yaml:"responseSelection,omitempty"`
	URL              string         `json:"url" yaml:"url"`
	HTTPVerb         string         `json:"httpVerb" yaml:"httpVerb"`
}

func (gq *GraphQL) GetCursorJSONPath() (string, bool) {
	if gq.Cursor == nil {
		return "", false
	}
	return gq.Cursor.getJSONPath()
}

func (gq *GraphQL) GetResponseJSONPath() (string, bool) {
	if gq.Cursor == nil {
		return "", false
	}
	return gq.ReponseSelection.getJSONPath()
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
