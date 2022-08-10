package openapistackql

const (
	MethodDescription string = "description"
	MethodName        string = "MethodName"
	RequiredParams    string = "RequiredParams"
	SQLVerb           string = "SQLVerb"
)

const (
	ExtensionKeyAlwaysRequired      string = "x-alwaysRequired"
	ExtensionKeyGraphQL             string = "x-stackQL-graphQL"
	ExtensionKeyQueryParamTranspose string = "x-stackQL-queryParamTranspose"
	ExtensionKeyProvider            string = "x-stackql-provider"
	ExtensionKeyResources           string = "x-stackQL-resources"
)

const (
	RequestBodyKeyPrefix    string = "data"
	RequestBodyKeyDelimiter string = "__"
	RequestBodyBaseKey      string = RequestBodyKeyPrefix + RequestBodyKeyDelimiter
)
