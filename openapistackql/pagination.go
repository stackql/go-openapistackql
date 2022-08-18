package openapistackql

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-openapi/jsonpointer"
)

var (
	linksNextRegex *regexp.Regexp = regexp.MustCompile(`.*<(?P<nextURL>[^>]*)>;\ rel="next".*`)
)

type Pagination struct {
	RequestToken  *TokenSemantic `json:"requestToken,omitempty" yaml:"requestToken,omitempty"`
	ResponseToken *TokenSemantic `json:"responseToken,omitempty" yaml:"responseToken,omitempty"`
}

var _ jsonpointer.JSONPointable = (Pagination)(Pagination{})

func (qt Pagination) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "requestToken":
		return qt.RequestToken, nil
	case "responseToken":
		return qt.ResponseToken, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from QueryTranspose doc object", token)
	}
}

type TokenTransformer func(interface{}) (interface{}, error)

type TransformerLocator interface {
	GetTransformer(tokenSemantic *TokenSemantic) (TokenTransformer, error)
}

type StandardTransformerLocator struct{}

func NewStandardTransformerLocator() TransformerLocator {
	return &StandardTransformerLocator{}
}

func (stl *StandardTransformerLocator) GetTransformer(tokenSemantic *TokenSemantic) (TokenTransformer, error) {
	switch strings.ToLower(tokenSemantic.Location) {
	case "header":
		return getHeaderTransformer(tokenSemantic)
	default:
		return nil, nil
	}
}

func getHeaderTransformer(tokenSemantic *TokenSemantic) (TokenTransformer, error) {
	if tokenSemantic.Algorithm == "" && strings.ToLower(tokenSemantic.Key) == "link" && strings.ToLower(tokenSemantic.Location) == "header" {
		return defaultLinkHeaderTransformer, nil
	}

	return func(input interface{}) (interface{}, error) {
		h, ok := input.(http.Header)
		if !ok {
			return nil, fmt.Errorf("cannot ingest purported http header of type = '%T'", h)
		}
		s := h.Values(tokenSemantic.Key)
		resArr := linksNextRegex.FindStringSubmatch(strings.Join(s, ","))
		if len(resArr) == 2 {
			return resArr[1], nil
		}
		return "", nil
	}, nil
}

func DefaultLinkHeaderTransformer(input interface{}) (interface{}, error) {
	return defaultLinkHeaderTransformer(input)
}

func defaultLinkHeaderTransformer(input interface{}) (interface{}, error) {
	h, ok := input.(http.Header)
	if !ok {
		return nil, fmt.Errorf("cannot ingest purported http header of type = '%T'", h)
	}
	s := h.Values("Link")
	resArr := linksNextRegex.FindStringSubmatch(strings.Join(s, ","))
	if len(resArr) == 2 {
		return resArr[1], nil
	}
	return "", nil
}
