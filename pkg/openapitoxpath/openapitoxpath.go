package openapitoxpath

import (
	"strings"
)

func ToXpath(path []string) string {
	return strings.Join(path, "/")
}

func ToPathSlice(path string) []string {
	return strings.Split(path, "/")
}
