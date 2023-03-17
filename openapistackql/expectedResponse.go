package openapistackql

var (
	_ ExpectedResponse = &standardExpectedResponse{}
)

type ExpectedResponse interface {
	GetBodyMediaType() string
	GetOpenAPIDocKey() string
	GetObjectKey() string
	GetSchema() Schema
	//
	setSchema(Schema)
	setBodyMediaType(string)
}

type standardExpectedResponse struct {
	BodyMediaType string `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
	OpenAPIDocKey string `json:"openAPIDocKey,omitempty" yaml:"openAPIDocKey,omitempty"`
	ObjectKey     string `json:"objectKey,omitempty" yaml:"objectKey,omitempty"`
	Schema        Schema
}

func (er *standardExpectedResponse) setBodyMediaType(s string) {
	er.BodyMediaType = s
}

func (er *standardExpectedResponse) setSchema(s Schema) {
	er.Schema = s
}

func (er *standardExpectedResponse) GetBodyMediaType() string {
	return er.BodyMediaType
}

func (er *standardExpectedResponse) GetOpenAPIDocKey() string {
	return er.OpenAPIDocKey
}

func (er *standardExpectedResponse) GetObjectKey() string {
	return er.ObjectKey
}

func (er *standardExpectedResponse) GetSchema() Schema {
	return er.Schema
}
