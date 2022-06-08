package openapistackql

import "net/http"

type Response struct {
	rawBody       interface{}
	processedBody interface{}
	httpResponse  *http.Response
}

func (r *Response) GetHttpResponse() *http.Response {
	return r.httpResponse
}

func (r *Response) GetBody() interface{} {
	return r.rawBody
}

func (r *Response) GetProcessedBody() interface{} {
	return r.processedBody
}

func NewResponse(processedBody, rawBody interface{}, r *http.Response) *Response {
	return &Response{
		processedBody: processedBody,
		rawBody:       rawBody,
		httpResponse:  r,
	}
}
