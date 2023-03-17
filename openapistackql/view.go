package openapistackql

import (
	"fmt"
	"regexp"

	"github.com/go-openapi/jsonpointer"
)

var (
	sqlDialectRegex *regexp.Regexp            = regexp.MustCompile(`sqlDialect(?:\s)*==(?:\s)*"(?P<sqlDialect>[^<>"\s]*)"`)
	_               View                      = &standardView{}
	_               jsonpointer.JSONPointable = standardView{}
)

type View interface {
	GetDDLForSqlDialect(sqlBackend string) (string, bool)
}

func GetTestingView() standardView {
	return standardView{}
}

type standardView struct {
	Predicate string        `json:"predicate" yaml:"predicate"`
	DDL       string        `json:"ddl" yaml:"ddl"`
	Fallback  *standardView `json:"fallback" yaml:"fallback"` // Future proofing for predicate failover
}

func (v *standardView) getSqlDialectName() string {
	inputString := v.Predicate
	for i, name := range sqlDialectRegex.SubexpNames() {
		if name == "sqlDialect" {
			submatches := sqlDialectRegex.FindStringSubmatch(inputString)
			if len(submatches) > i {
				return submatches[i]
			}
		}
	}
	return ""
}

func (v *standardView) GetDDLForSqlDialect(sqlBackend string) (string, bool) {
	sqlBackendAccepted := v.getSqlDialectName()
	if sqlBackendAccepted == "" {
		return v.DDL, true
	}
	if sqlBackendAccepted == sqlBackend {
		return v.DDL, true
	}
	if v.Fallback != nil {
		return v.Fallback.GetDDLForSqlDialect(sqlBackend)
	}
	return "", false

}

func (qt standardView) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "ddl":
		return qt.DDL, nil
	case "predicate":
		return qt.Predicate, nil
	case "fallback":
		return qt.Fallback, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from View doc object", token)
	}
}
