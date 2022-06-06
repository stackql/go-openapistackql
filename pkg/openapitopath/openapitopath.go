package openapitopath

import (
	"strings"
)

type PathResolver interface {
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

func (xpr *XPathResolver) ToPathSlice(path string) []string {
	strSlice := strings.Split(path, "/")
	if len(strSlice) > 0 && strSlice[0] == "" {
		strSlice = strSlice[1:]
	}
	return strSlice
}

func (jpr *JSONPathResolver) ToPathSlice(path string) []string {
	strSlice := strings.Split(path, ".")
	if len(strSlice) > 0 && strSlice[0] == "$" {
		strSlice = strSlice[1:]
	}
	var rv []string
	for _, s := range strSlice {
		rv = append(rv, strings.TrimSuffix(s, "[*]"))
		if strings.TrimSuffix(s, "[*]") != s {
			rv = append(rv, "[*]")
		}
	}
	return rv
}
