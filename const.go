package openapistackql

const (
	MethodDescription string = "description"
	MethodName        string = "MethodName"
	RequiredParams    string = "RequiredParams"
)

const (
	ExtensionKeyAlwaysRequired string = "x-alwaysRequired"
	ExtensionKeyProvider       string = "x-stackql-provider"
	ExtensionKeyResources      string = "x-stackQL-resources"
)

const (
	RequestBodyKeyPrefix    string = "data"
	RequestBodyKeyDelimiter string = "__"
	RequestBodyBaseKey      string = RequestBodyKeyPrefix + RequestBodyKeyDelimiter
)
