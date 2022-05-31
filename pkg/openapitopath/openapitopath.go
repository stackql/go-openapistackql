package openapitopath

import (
	"strings"
)

type PathResolver interface {
	// ToPath(path []string) string
	ToPathSlice(path string) []string
}

type XPathResolver struct {
	_ struct{}
}
type JSONPathResolver struct {
	_ struct{}
}

func NewXPathResolver() PathResolver {
	return &XPathResolver{}
}

func NewJSONPathResolver() PathResolver {
	return &JSONPathResolver{}
}

// func (xpr *XPathResolver) ToPath(path []string) string {
// 	return strings.Join(path, "/")
// }

func (xpr *XPathResolver) ToPathSlice(path string) []string {
	strSlice := strings.Split(path, "/")
	if len(strSlice) > 0 && strSlice[0] == "" {
		strSlice = strSlice[1:]
	}
	return strSlice
}

// func (jpr *JSONPathResolver) ToPath(path []string) string {
// 	sj := strings.Join(path, ".")
// 	return "$." + sj
// }

func (jpr *JSONPathResolver) ToPathSlice(path string) []string {
	strSlice := strings.Split(path, ".")
	if len(strSlice) > 0 && strSlice[0] == "$" {
		strSlice = strSlice[1:]
	}
	return strSlice
}
