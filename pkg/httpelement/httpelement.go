package httpelement

import (
	"fmt"
	"strings"
)

type HTTPElementLocation int
type FlowControlAction int

const (
	QueryParam HTTPElementLocation = iota
	PathParam
	Header
	BodyAttribute
)

const (
	IncrementRequest FlowControlAction = iota
	ConcludeSuccess
)

const (
	QueryStr         string = "query"
	PathStr          string = "path"
	HeaderStr        string = "header"
	BodyAttributeStr string = "body"
)

type HTTPElement interface {
	GetName() string
	GetLocation() HTTPElementLocation
}

func NewHTTPElement(name, location string) (HTTPElement, error) {
	switch strings.ToLower(location) {
	case QueryStr:
		return &IHTTPElement{
			elementType: QueryParam,
			name:        name,
		}, nil
	case PathStr:
		return &IHTTPElement{
			elementType: PathParam,
			name:        name,
		}, nil
	case HeaderStr:
		return &IHTTPElement{
			elementType: Header,
			name:        name,
		}, nil
	case BodyAttributeStr:
		return &IHTTPElement{
			elementType: BodyAttribute,
			name:        name,
		}, nil
	default:
		return nil, fmt.Errorf("cannot accomodate HTTP element location = '%s'", location)
	}
}

type IHTTPElement struct {
	_           struct{}
	elementType HTTPElementLocation
	name        string
}

func (he *IHTTPElement) GetName() string {
	return he.name
}

func (he *IHTTPElement) GetLocation() HTTPElementLocation {
	return he.elementType
}
