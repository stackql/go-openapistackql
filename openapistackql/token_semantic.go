package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"

	"github.com/stackql/go-openapistackql/pkg/httpelement"
	"github.com/stackql/go-openapistackql/pkg/response"
)

var (
	_ TokenSemantic             = &standardTokenSemantic{}
	_ jsonpointer.JSONPointable = (TokenSemantic)(&standardTokenSemantic{})
)

type TokenSemanticArgs map[string]interface{}

type TokenSemantic interface {
	JSONLookup(token string) (interface{}, error)
	GetAlgorithm() string
	GetArgs() TokenSemanticArgs
	GetKey() string
	GetLocation() string
	GetTransformer() (TokenTransformer, error)
	GetProcessedToken(res response.Response) (interface{}, error)
}

type standardTokenSemantic struct {
	Algorithm string            `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	Args      TokenSemanticArgs `json:"args,omitempty" yaml:"args,omitempty"`
	Key       string            `json:"key,omitempty" yaml:"key,omitempty"`
	Location  string            `json:"location,omitempty" yaml:"location,omitempty"`
}

func (ts *standardTokenSemantic) GetTransformer() (TokenTransformer, error) {
	return ts.getTransformer()
}

func (ts *standardTokenSemantic) getTransformer() (TokenTransformer, error) {
	tl := NewStandardTransformerLocator()
	return tl.GetTransformer(ts)
}

func (ts *standardTokenSemantic) GetProcessedToken(res response.Response) (interface{}, error) {
	return ts.getProcessedToken(res)
}

func (ts *standardTokenSemantic) getProcessedToken(res response.Response) (interface{}, error) {
	rawToken, err := ts.getRawToken(res)
	if err != nil {
		return nil, err
	}
	transformer, transformerErr := ts.getTransformer()
	if transformerErr != nil {
		return nil, transformerErr
	}
	processedToken, processedTokenErr := transformer(rawToken)
	if processedTokenErr != nil {
		return nil, processedTokenErr
	}
	return processedToken, nil
}

func (ts *standardTokenSemantic) getRawToken(res response.Response) (interface{}, error) {
	httpElement, err := httpelement.NewHTTPElement(
		ts.Key,
		ts.Location,
	)
	if err != nil {
		return nil, err
	}
	return res.ExtractElement(httpElement)
}

func (ts *standardTokenSemantic) GetAlgorithm() string {
	return ts.Algorithm
}

func (ts *standardTokenSemantic) GetArgs() TokenSemanticArgs {
	return ts.Args
}

func (ts *standardTokenSemantic) GetKey() string {
	return ts.Key
}

func (ts *standardTokenSemantic) GetLocation() string {
	return ts.Location
}

func (qt *standardTokenSemantic) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "algorithm":
		return qt.Algorithm, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from TokenSemantic doc object", token)
	}
}

func (tsa TokenSemanticArgs) GetRegex() (string, bool) {
	r, ok := tsa["regex"]
	if !ok {
		return "", false
	}
	rv, ok := r.(string)
	if !ok {
		return "", false
	}
	return rv, true
}
