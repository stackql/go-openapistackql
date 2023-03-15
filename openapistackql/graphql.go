package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

var (
	_ jsonpointer.JSONPointable = standardGraphQL{}
	_ GraphQL                   = &standardGraphQL{}
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

type GraphQL interface {
	JSONLookup(token string) (interface{}, error)
	GetCursorJSONPath() (string, bool)
	GetResponseJSONPath() (string, bool)
	GetID() string
	GetQuery() string
	GetURL() string
	GetHTTPVerb() string
	GetCursor() GraphQLElement
	GetResponseSelection() GraphQLElement
}

type standardGraphQL struct {
	ID               string         `json:"id" yaml:"id"`
	Query            string         `json:"query,omitempty" yaml:"query,omitempty"` // Required
	Cursor           GraphQLElement `json:"cursor,omitempty" yaml:"cursor,omitempty"`
	ReponseSelection GraphQLElement `json:"responseSelection,omitempty" yaml:"responseSelection,omitempty"`
	URL              string         `json:"url" yaml:"url"`
	HTTPVerb         string         `json:"httpVerb" yaml:"httpVerb"`
}

func (gq *standardGraphQL) GetCursorJSONPath() (string, bool) {
	if gq.Cursor == nil {
		return "", false
	}
	return gq.Cursor.getJSONPath()
}

func (gq *standardGraphQL) GetResponseJSONPath() (string, bool) {
	if gq.Cursor == nil {
		return "", false
	}
	return gq.ReponseSelection.getJSONPath()
}

func (gq standardGraphQL) JSONLookup(token string) (interface{}, error) {
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

func (gq *standardGraphQL) GetID() string {
	return gq.ID
}

func (gq *standardGraphQL) GetQuery() string {
	return gq.Query
}

func (gq *standardGraphQL) GetURL() string {
	return gq.URL
}

func (gq *standardGraphQL) GetHTTPVerb() string {
	return gq.HTTPVerb
}

func (gq *standardGraphQL) GetCursor() GraphQLElement {
	return gq.Cursor
}

func (gq *standardGraphQL) GetResponseSelection() GraphQLElement {
	return gq.ReponseSelection
}
