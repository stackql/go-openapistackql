package urltranslate

import (
	"fmt"
	"strings"
)

type QueryElement interface {
	isQueryElement()
	String() string
}

type QueryVar interface {
	QueryElement
	GetName() string
	FullString() string
	// GetRegexpStr() string
	// IsRegexp() bool
}

type varWithRegexp struct {
	_                    struct{}
	raw, name, regexpStr string
}

type stringFragment struct {
	_   struct{}
	raw string
}

func newStringFragment(s string) QueryElement {
	return &stringFragment{
		raw: s,
	}
}

func (sf *stringFragment) isQueryElement() {}

func (sf *stringFragment) String() string {
	return sf.raw
}

func (vwr *varWithRegexp) isQueryElement() {}

func (vwr *varWithRegexp) String() string {
	return fmt.Sprintf("{%s}", vwr.name)
}

// func (vwr *varWithRegexp) IsRegexp() bool {
// 	return vwr.regexpStr != ""
// }

// func (vwr *varWithRegexp) GetRegexpStr() string {
// 	return vwr.regexpStr
// }

func (vwr *varWithRegexp) GetName() string {
	return vwr.name
}

func (vwr *varWithRegexp) FullString() string {
	return fmt.Sprintf("{%s}", vwr.raw)
}

type ParameterisedURL interface {
	Raw() string
	String() string
	GetVarByName(name string) (QueryVar, bool)
}

type urlWithParams struct {
	raw    string
	parsed []QueryElement
}

func (uwp *urlWithParams) Raw() string {
	return uwp.raw
}

func (uwp *urlWithParams) String() string {
	var sb strings.Builder
	for _, elem := range uwp.parsed {
		sb.WriteString(elem.String())
	}
	return sb.String()
}

func (uwp *urlWithParams) GetVarByName(name string) (QueryVar, bool) {
	for _, qe := range uwp.parsed {
		switch qe := qe.(type) {
		case QueryVar:
			if qe.GetName() == name {
				return qe, true
			}
		}
	}
	return nil, false
}

func extractRegexpVariable(v string) (QueryVar, error) {
	splitVar := strings.SplitN(v, ":", 2)
	switch len(splitVar) {
	case 1:
		return &varWithRegexp{
			raw:  v,
			name: v,
		}, nil
	case 2:
		rv := &varWithRegexp{
			raw:       v,
			name:      splitVar[0],
			regexpStr: splitVar[1],
		}
		return rv, nil
	default:
		return nil, fmt.Errorf("unnaceptable variable name '%s' with %d colons", v, len(splitVar))
	}
}

func ExtractParameterisedURL(s string) (ParameterisedURL, error) {
	return extractParameterisedURL(s)
}

func extractParameterisedURL(s string) (ParameterisedURL, error) {
	rv := &urlWithParams{
		raw: s,
	}
	defaultError := fmt.Errorf("could not extract parameterised URL from string = '%s'", s)
	var inVar bool
	var sb strings.Builder
	for i := range s {
		c := s[i]
		if c == '{' {
			if inVar {
				return nil, defaultError
			}
			inVar = true
			if sb.Len() > 0 {
				rv.parsed = append(rv.parsed, newStringFragment(sb.String()))
			}
			sb.Reset()
			continue
		}
		if c == '}' {
			if !inVar {
				return nil, defaultError
			}
			inVar = false
			if sb.Len() > 0 {
				v, err := extractRegexpVariable(sb.String())
				if err != nil {
					return nil, err
				}
				rv.parsed = append(rv.parsed, v)
			}
			sb.Reset()
			continue
		}
		sb.WriteByte(c)
	}
	if inVar {
		return nil, defaultError
	}
	if sb.Len() > 0 {
		rv.parsed = append(rv.parsed, newStringFragment(sb.String()))
	}
	return rv, nil
}

func SanitiseServerURL(s string) (string, error) {
	pu, err := extractParameterisedURL(s)
	if err != nil {
		return "", err
	}
	return pu.String(), err
}
