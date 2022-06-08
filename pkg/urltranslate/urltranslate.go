package urltranslate

import (
	"fmt"
	"strconv"
	"strings"
)

type QueryElement interface {
	isQueryElement()
	IsVariable() bool
	String() string
	FullString() string
}

type QueryVar interface {
	QueryElement
	GetName() string
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

func (sf *stringFragment) IsVariable() bool {
	return false
}

func (sf *stringFragment) String() string {
	return sf.raw
}

func (sf *stringFragment) FullString() string {
	return sf.raw
}

func (vwr *varWithRegexp) isQueryElement() {}

func (vwr *varWithRegexp) IsVariable() bool {
	return true
}

func (vwr *varWithRegexp) String() string {
	return fmt.Sprintf("{%s}", vwr.name)
}

func (vwr *varWithRegexp) GetName() string {
	return vwr.name
}

func (vwr *varWithRegexp) FullString() string {
	return fmt.Sprintf("{%s}", vwr.raw)
}

type ParameterisedURL interface {
	Raw() string
	String() string
	GetElementByString(s string) (QueryElement, bool)
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

func (uwp *urlWithParams) GetElementByString(s string) (QueryElement, bool) {
	isVar := strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
	if isVar {
		varName := strings.TrimSuffix(strings.TrimPrefix(s, "{"), "}")
		varVal, ok := uwp.GetVarByName(varName)
		return varVal, ok
	}
	return newStringFragment(s), strings.Contains(uwp.raw, s)
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

type URLHost interface {
	GetHost() string
}

type URLHostSimple struct {
	_         struct{}
	raw, host string
	port      int
}

func (h *URLHostSimple) GetHost() string {
	return h.host
}

func ParseURLHost(h string) (URLHost, error) {
	hSplit := strings.Split(h, ":")
	switch len(hSplit) {
	case 1:
		return &URLHostSimple{
			raw:  h,
			host: h,
			port: -1,
		}, nil
	case 2:
		port, err := strconv.Atoi(hSplit[1])
		if err != nil {
			return nil, err
		}
		return &URLHostSimple{
			raw:  h,
			host: hSplit[0],
			port: port,
		}, nil
	default:
		return nil, fmt.Errorf("cannot parse URL host from string '%s'", h)
	}
}
