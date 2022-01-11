package openapistackql

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/jsoninfo"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
)

type OperationRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *openapi3.Operation
}

type PathItemRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *openapi3.PathItem
}

type ServiceRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *Service
}

var _ jsonpointer.JSONPointable = (*OperationRef)(nil)

func (value *OperationRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

func (value *OperationRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

func (value *OperationRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

func (value OperationRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

func foundUnresolvedRef(ref string) error {
	return fmt.Errorf("found unresolved ref: %q", ref)
}
