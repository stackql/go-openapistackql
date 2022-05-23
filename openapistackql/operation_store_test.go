package openapistackql_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	. "github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/go-openapistackql/test/pkg/testutil"

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

func TestXMLHandle(t *testing.T) {
	setupFileRoot(t)
	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"text/xml"}},
		StatusCode: 200,
		Body:       testutil.GetAwsEc2ListMultiResponseReader(),
	}

	b, err := GetServiceDocBytes(fmt.Sprintf("aws/%s/services/ec2.yaml", "v0.1.0"))
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)

	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	assert.Equal(t, svc.GetName(), "ec2")

	rsc, err := svc.GetResource("volumes")
	assert.NilError(t, err)
	assert.Assert(t, rsc != nil)

	ops, st, ok := rsc.GetFirstMethodFromSQLVerb("select")
	assert.Assert(t, ok)
	assert.Assert(t, st != "")
	assert.Assert(t, ops != nil)

	processedResponse, err := ops.ProcessResponse(res)
	assert.NilError(t, err)
	assert.Assert(t, processedResponse != nil)

	// m, err := GetSubObjTyped(getAwsEc2ListMultiResponseReader(), "/DescribeVolumesResponse/volumeSet/item", sc)

	mc, ok := processedResponse.([]map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(mc) == 2)
	assert.Assert(t, mc[1]["iops"] == 100)
	assert.Assert(t, mc[1]["size"] == 8)

}
