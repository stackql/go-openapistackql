package openapistackql_test

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	. "github.com/stackql/go-openapistackql/openapistackql"

	"gotest.tools/assert"
)

func TestPlaceholder(t *testing.T) {
	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(`{"a": { "b": [ "c" ] } }`)),
	}
	s := NewSchema(openapi3.NewSchema(), "")
	pr, err := s.ProcessHttpResponse(res, "")
	assert.NilError(t, err)
	assert.Assert(t, pr != nil)
}
