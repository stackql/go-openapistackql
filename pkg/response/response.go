package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PaesslerAG/jsonpath"
	"github.com/antchfx/xmlquery"
	"github.com/stackql/go-openapistackql/pkg/httpelement"
	"github.com/stackql/go-openapistackql/pkg/media"
	"github.com/stackql/go-openapistackql/pkg/xmlmap"
)

var (
	_ Response = &basicResponse{}
)

type Response interface {
	GetHttpResponse() *http.Response
	GetBody() interface{}
	GetProcessedBody() interface{}
	ExtractElement(e httpelement.HTTPElement) (interface{}, error)
	Error() string
	String() string
}

type basicResponse struct {
	_             struct{}
	rawBody       interface{}
	processedBody interface{}
	httpResponse  *http.Response
	bodyMediaType string
}

func (r *basicResponse) GetHttpResponse() *http.Response {
	return r.httpResponse
}

func (r *basicResponse) GetBody() interface{} {
	return r.rawBody
}

func (r *basicResponse) GetProcessedBody() interface{} {
	return r.processedBody
}

func (r *basicResponse) String() string {
	return r.string()
}

func (r *basicResponse) string() string {
	var baseString string
	switch body := r.processedBody.(type) {
	case map[string]interface{}:
		b, err := json.Marshal(body)
		if err == nil {
			baseString = string(b)
		}
	case map[string]string:
		b, err := json.Marshal(body)
		if err == nil {
			baseString = string(b)
		}
	}
	if r.httpResponse != nil {
		if baseString != "" {
			return fmt.Sprintf(`{ "statusCode": %d, "body": %s  }`, r.httpResponse.StatusCode, baseString)
		}
	}
	if baseString != "" {
		return fmt.Sprintf(`{ "body": %s  }`, baseString)
	}
	return ""
}

func (r *basicResponse) Error() string {
	baseString := r.string()
	if baseString != "" {
		return fmt.Sprintf(`{ "httpError": %s }`, baseString)
	}
	return `{ "httpError": { "message": "unknown error" } }`
}

func (r *basicResponse) ExtractElement(e httpelement.HTTPElement) (interface{}, error) {
	elementLocation := e.GetLocation()
	switch elementLocation {
	case httpelement.BodyAttribute:
		// refactor heaps of shit here
		switch body := r.rawBody.(type) {
		case *xmlquery.Node:
			elem, err := xmlmap.GetSubObjFromNode(body, e.GetName())
			return elem, err
		default:
			processedResponse, err := jsonpath.Get(e.GetName(), body)
			return processedResponse, err
		}
	case httpelement.Header:
		return r.httpResponse.Header.Values(e.GetName()), nil
	default:
		return nil, fmt.Errorf("http element type '%v' not supported", elementLocation)
	}
}

func NewResponse(processedBody, rawBody interface{}, r *http.Response) Response {
	mt, _ := media.GetResponseMediaType(r, "")
	return &basicResponse{
		processedBody: processedBody,
		rawBody:       rawBody,
		httpResponse:  r,
		bodyMediaType: mt,
	}
}
