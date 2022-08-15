package openapistackql

type NamedSchema struct {
	s          *Schema
	name       string
	location   string
	isRequired bool
}

func (ns *NamedSchema) GetLocation() string {
	return ns.location
}

func (ns *NamedSchema) GetName() string {
	return ns.name
}

func (ns *NamedSchema) GetSchema() (*Schema, bool) {
	return ns.s, true
}

func (ns *NamedSchema) GetType() string {
	return ns.s.Type
}

func (ns *NamedSchema) IsRequired() bool {
	return ns.isRequired
}

func (ns *NamedSchema) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(ns.s.Type, lhs, rhs)
}

func NewAddressableRequestBodyProperty(name string, s *Schema, isRequired bool) Addressable {
	return &NamedSchema{
		s:          s,
		name:       name,
		location:   "requestBody",
		isRequired: isRequired,
	}
}

type Addressable interface {
	ConditionIsValid(lhs string, rhs interface{}) bool
	GetLocation() string
	GetName() string
	GetSchema() (*Schema, bool)
	GetType() string
	IsRequired() bool
}
