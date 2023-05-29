package openapistackql

import "fmt"

var (
	_ Addressable = &namedSchema{}
)

type namedSchema struct {
	s          Schema
	name       string
	location   string
	isRequired bool
}

func (ns *namedSchema) GetLocation() string {
	return ns.location
}

func (ns *namedSchema) GetName() string {
	return ns.name
}

func (ns *namedSchema) GetSchema() (Schema, bool) {
	return ns.s, true
}

func (ns *namedSchema) GetType() string {
	return ns.s.GetType()
}

func (ns *namedSchema) IsRequired() bool {
	return ns.isRequired
}

func (ns *namedSchema) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(ns.s.GetType(), lhs, rhs)
}

func NewRequiredAddressableRequestBodyProperty(name string, s Schema) Addressable {
	return newAddressableRequestBodyProperty(name, s, true)
}

func NewOptionalAddressableRequestBodyProperty(name string, s Schema) Addressable {
	return newAddressableRequestBodyProperty(name, s, false)
}

func newAddressableRequestBodyProperty(name string, s Schema, isRequired bool) Addressable {
	return &namedSchema{
		s:          s,
		name:       name,
		location:   "requestBody",
		isRequired: isRequired,
	}
}

func newAddressableServerVariable(name string, s Schema, isRequired bool) Addressable {
	return &namedSchema{
		s:          s,
		name:       name,
		location:   "server",
		isRequired: isRequired,
	}
}

type Addressable interface {
	ConditionIsValid(lhs string, rhs interface{}) bool
	GetLocation() string
	GetName() string
	GetSchema() (Schema, bool)
	GetType() string
	IsRequired() bool
}

func DefaultRequestBodyAttributeRename(k string) string {
	return defaultRequestBodyAttributeRename(k)
}

func defaultRequestBodyAttributeRename(k string) string {
	return fmt.Sprintf("%s%s", RequestBodyBaseKey, k)
}
