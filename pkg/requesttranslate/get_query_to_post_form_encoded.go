package requesttranslate

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func NewGetQueryToPostFormEncodedTranslator(byteEncoding string) RequestTranslator {
	return &GetQueryToPostFormEncodedTranslator{
		byteEncoding: byteEncoding,
	}
}

type GetQueryToPostFormEncodedTranslator struct {
	byteEncoding string
}

func (gp *GetQueryToPostFormEncodedTranslator) Translate(req *http.Request) (*http.Request, error) {
	rv := req.Clone(req.Context())
	if req.URL == nil {
		return nil, fmt.Errorf("cannot translate nil URL")
	}
	if req.Body != nil {
		return nil, fmt.Errorf("cannot translate GET query params to POST form-encoded where GET body is not nil")
	}
	rq := req.URL.RawQuery
	if rq != "" {
		rv.Body = io.NopCloser(bytes.NewBufferString(rq))
	}
	rv.URL.RawQuery = ""
	rv.Method = http.MethodPost
	rv.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	return rv, nil
}
