package openapistackql

var (
	_ HTTPArmoury = &standardHTTPArmoury{}
)

type HTTPArmoury interface {
	AddRequestParams(HTTPArmouryParameters)
	GetRequestParams() []HTTPArmouryParameters
	GetRequestSchema() Schema
	GetResponseSchema() Schema
	SetRequestParams([]HTTPArmouryParameters)
	SetRequestSchema(Schema)
	SetResponseSchema(Schema)
}

type standardHTTPArmoury struct {
	RequestParams  []HTTPArmouryParameters
	RequestSchema  Schema
	ResponseSchema Schema
}

func (ih *standardHTTPArmoury) GetRequestParams() []HTTPArmouryParameters {
	return ih.RequestParams
}

func (ih *standardHTTPArmoury) SetRequestParams(ps []HTTPArmouryParameters) {
	ih.RequestParams = ps
}

func (ih *standardHTTPArmoury) AddRequestParams(p HTTPArmouryParameters) {
	ih.RequestParams = append(ih.RequestParams, p)
}

func (ih *standardHTTPArmoury) SetRequestSchema(s Schema) {
	ih.RequestSchema = s
}

func (ih *standardHTTPArmoury) SetResponseSchema(s Schema) {
	ih.ResponseSchema = s
}

func (ih *standardHTTPArmoury) GetRequestSchema() Schema {
	return ih.RequestSchema
}

func (ih *standardHTTPArmoury) GetResponseSchema() Schema {
	return ih.ResponseSchema
}

func NewHTTPArmoury() HTTPArmoury {
	return &standardHTTPArmoury{}
}
