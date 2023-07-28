package openapistackql

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/jsoninfo"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
)

type OperationRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *openapi3.Operation
}

func (opr OperationRef) ExtractPathItem() string {
	return opr.extractPathItem()
}

func (opr OperationRef) extractPathItem() string {
	s := opr.extractFragment()
	elems := strings.Split(strings.TrimPrefix(s, "/paths/"), "/")
	toUse := elems
	if len(elems) > 1 {
		toUse = elems[0 : len(elems)-1]
	}
	s2 := strings.Join(toUse, "/")
	return strings.ReplaceAll(s2, "~1", "/")
}

func (opr OperationRef) ExtractMethodItem() string {
	return opr.extractMethodItem()
}

func (opr OperationRef) extractMethodItem() string {
	return extractSuffix(opr.Ref)
}

func (opr OperationRef) ExtractServiceDocPath() string {
	return opr.extractServiceDocPath()
}

func (opr OperationRef) extractServiceDocPath() string {
	s := opr.Ref
	elems := strings.Split(s, "#")
	if len(elems) > 1 {
		return elems[0]
	}
	return s
}

func extractFragment(s string) string {
	if strings.HasPrefix(s, "#") {
		return s[1:]
	}
	elems := strings.Split(s, "#")
	if len(elems) > 2 {
		return strings.Join(elems[2:], "#")
	}
	return elems[len(elems)-1]
}

func extractSuffix(s string) string {
	sf := extractFragment(s)
	elems := strings.Split(sf, "/")
	return elems[len(elems)-1]
}

func (opr OperationRef) extractFragment() string {
	return extractFragment(opr.Ref)
}

type OperationStoreRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *standardOperationStore
}

func (osr *OperationStoreRef) hasValue() bool {
	return osr.Value != nil
}

func (osr *OperationStoreRef) extractMethodItem() string {
	return extractSuffix(osr.Ref)
}

type PathItemRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *openapi3.PathItem
}

type ServiceRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *standardService
}

type ResourcesRef struct {
	Ref   string `json:"$ref" yaml:"$ref"`
	Value *standardResourceRegister
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

var _ jsonpointer.JSONPointable = (*OperationStoreRef)(nil)

func (value *OperationStoreRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

func (value *OperationStoreRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// func (value *OperationStoreRef) Validate(ctx context.Context) error {
// 	if v := value.Value; v != nil {
// 		return v.Validate(ctx)
// 	}
// 	return foundUnresolvedRef(value.Ref)
// }

func (value OperationStoreRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

func foundUnresolvedRef(ref string) error {
	return fmt.Errorf("found unresolved ref: %q", ref)
}
