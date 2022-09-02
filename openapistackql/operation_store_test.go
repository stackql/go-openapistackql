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

var (
	dummmyContrivedProv *Provider = &Provider{
		Name: "github",
	}
	dummmyGoogleProv *Provider = &Provider{
		Name: "google",
	}
	dummmyK8sProv *Provider = &Provider{
		Name: "k8s",
	}
)

func TestPlaceholder(t *testing.T) {
	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(`{"a": { "b": [ "c" ] } }`)),
	}
	s := NewSchema(openapi3.NewSchema(), nil, "")
	pr, err := s.ProcessHttpResponseTesting(res, "", "")
	assert.NilError(t, err)
	assert.Assert(t, pr != nil)
}

func TestXPathHandle(t *testing.T) {
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

	mc, ok := processedResponse.GetProcessedBody().([]map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(mc) == 2)
	assert.Assert(t, mc[1]["iops"] == 100)
	assert.Assert(t, mc[1]["size"] == 8)

}

func TestJSONPathHandle(t *testing.T) {
	setupFileRoot(t)

	rdr, err := testutil.GetK8SNodesListMultiResponseReader()

	assert.NilError(t, err)

	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       rdr,
	}

	b, err := GetServiceDocBytes(fmt.Sprintf("k8s/%s/services/core_v1.yaml", "v0.1.0"))
	assert.NilError(t, err)

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)

	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	// assert.Equal(t, svc.GetName(), "ec2")

	rsc, err := svc.GetResource("node")
	assert.NilError(t, err)
	assert.Assert(t, rsc != nil)

	ops, _, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", nil)
	assert.Assert(t, ok)
	// assert.Assert(t, st != "")
	assert.Assert(t, ops != nil)

	processedResponse, err := ops.ProcessResponse(res)
	assert.NilError(t, err)
	assert.Assert(t, processedResponse != nil)

	mc, ok := processedResponse.GetProcessedBody().([]interface{})
	assert.Assert(t, ok)
	e0, ok := mc[0].(map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(mc) == 3)
	assert.Assert(t, e0["uid"] == "d5626684-69a3-4644-bf2b-a8e67bb44b01")

}

func TestXMLSchemaInterrogation(t *testing.T) {
	setupFileRoot(t)

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

	s, p, err := ops.GetSelectSchemaAndObjectPath()

	assert.NilError(t, err)
	assert.Assert(t, s != nil)
	assert.Assert(t, p != "")

	assert.Assert(t, s.GetName() == "Volume")

}

func TestVariableHostRouting(t *testing.T) {
	setupFileRoot(t)

	rdr, err := testutil.GetK8SNodesListMultiResponseReader()

	assert.NilError(t, err)

	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       rdr,
	}

	b, err := GetServiceDocBytes(fmt.Sprintf("k8s/%s/services/core_v1.yaml", "v0.1.0"))
	assert.NilError(t, err)

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)

	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	// assert.Equal(t, svc.GetName(), "ec2")

	rsc, err := svc.GetResource("node")
	assert.NilError(t, err)
	assert.Assert(t, rsc != nil)

	ops, _, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", nil)
	assert.Assert(t, ok)
	// assert.Assert(t, st != "")
	assert.Assert(t, ops != nil)

	processedResponse, err := ops.ProcessResponse(res)
	assert.NilError(t, err)
	assert.Assert(t, processedResponse != nil)

	mc, ok := processedResponse.GetProcessedBody().([]interface{})
	assert.Assert(t, ok)
	e0, ok := mc[0].(map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(mc) == 3)
	assert.Assert(t, e0["uid"] == "d5626684-69a3-4644-bf2b-a8e67bb44b01")

	params := NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{"cluster_addr": "k8shost"})
	assert.NilError(t, err)

	rvi, err := ops.Parameterize(dummmyK8sProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

	params = NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{"cluster_addr": "201.0.255.3"})
	assert.NilError(t, err)

	rvi, err = ops.Parameterize(dummmyK8sProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

}

func TestVariableHostRoutingFutureProofed(t *testing.T) {
	setupFileRoot(t)

	rdr, err := testutil.GetK8SNodesListMultiResponseReader()

	assert.NilError(t, err)

	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       rdr,
	}

	b, err := GetServiceDocBytes(fmt.Sprintf("k8s/%s/services/core_v1.yaml", "v0.1.1"))
	assert.NilError(t, err)

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)

	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	// assert.Equal(t, svc.GetName(), "ec2")

	rsc, err := svc.GetResource("node")
	assert.NilError(t, err)
	assert.Assert(t, rsc != nil)

	ops, _, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", nil)
	assert.Assert(t, ok)
	// assert.Assert(t, st != "")
	assert.Assert(t, ops != nil)

	processedResponse, err := ops.ProcessResponse(res)
	assert.NilError(t, err)
	assert.Assert(t, processedResponse != nil)

	mc, ok := processedResponse.GetProcessedBody().([]interface{})
	assert.Assert(t, ok)
	e0, ok := mc[0].(map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(mc) == 3)
	assert.Assert(t, e0["uid"] == "d5626684-69a3-4644-bf2b-a8e67bb44b01")

	params := NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{"cluster_addr": "k8shost"})
	assert.NilError(t, err)

	rvi, err := ops.Parameterize(dummmyK8sProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

	params = NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{"cluster_addr": "201.0.255.3"})
	assert.NilError(t, err)

	rvi, err = ops.Parameterize(dummmyK8sProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

}

func TestMethodLevelVariableHostRoutingFutureProofed(t *testing.T) {
	setupFileRoot(t)

	rdr, err := testutil.GetContrivedPagesResponseReader()

	assert.NilError(t, err)

	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       rdr,
	}

	b, err := GetServiceDocBytes(fmt.Sprintf("contrivedprovider/%s/services/contrived_service.yaml", "v0.1.0"))
	assert.NilError(t, err)

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)

	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	// assert.Equal(t, svc.GetName(), "ec2")

	rsc, err := svc.GetResource("pages")
	assert.NilError(t, err)
	assert.Assert(t, rsc != nil)

	stringParams := map[string]interface{}{
		"owner": "joeblow",
		"repo":  "dummyapp",
	}
	ops, _, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", stringParams)
	assert.Assert(t, ok)
	// assert.Assert(t, st != "")
	assert.Assert(t, ops != nil)

	processedResponse, err := ops.ProcessResponse(res)
	assert.NilError(t, err)
	assert.Assert(t, processedResponse != nil)

	mc, ok := processedResponse.GetProcessedBody().(map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, mc["url"] == "https://api.github.com/repos/dummyorg/dummyapp.io/pages")

	params := NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{
		"origin": "some.vanity.url.com",
		"owner":  "joeblow",
		"repo":   "dummyapp",
	})
	assert.NilError(t, err)

	rvi, err := ops.Parameterize(dummmyContrivedProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

	params = NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{
		"origin": "201.0.255.3",
		"owner":  "joeblow",
		"repo":   "dummyapp",
	})
	assert.NilError(t, err)

	rvi, err = ops.Parameterize(dummmyContrivedProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

}

func TestStaticHostRouting(t *testing.T) {
	setupFileRoot(t)

	rdr, err := testutil.GetGoogleFoldersListResponseReader()

	assert.NilError(t, err)

	res := &http.Response{
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		StatusCode: 200,
		Body:       rdr,
	}

	b, err := GetServiceDocBytes(fmt.Sprintf("googleapis.com/%s/services/cloudresourcemanager-v3.yaml", "v0.1.2"))
	assert.NilError(t, err)

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)

	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	// assert.Equal(t, svc.GetName(), "ec2")

	rsc, err := svc.GetResource("folders")
	assert.NilError(t, err)
	assert.Assert(t, rsc != nil)

	ops, _, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", map[string]interface{}{"parent": "organizations/123123123123"})
	assert.Assert(t, ok)
	// assert.Assert(t, st != "")
	assert.Assert(t, ops != nil)

	processedResponse, err := ops.ProcessResponse(res)
	assert.NilError(t, err)
	assert.Assert(t, processedResponse != nil)

	rm, ok := processedResponse.GetProcessedBody().(map[string]interface{})
	assert.Assert(t, ok)

	k := ops.GetSelectItemsKey()
	items, ok := rm[k]

	assert.Assert(t, ok)

	mc, ok := items.([]interface{})
	assert.Assert(t, ok)
	e0, ok := mc[0].(map[string]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(mc) == 1)
	assert.Assert(t, e0["name"] == "folders/12312312312")
	assert.Assert(t, e0["lifecycleState"] == "ACTIVE")

	params := NewHttpParameters(ops)
	err = params.IngestMap(map[string]interface{}{"parent": "organizations/123123123123"})
	assert.NilError(t, err)

	rvi, err := ops.Parameterize(dummmyGoogleProv, svc, params, nil)
	assert.NilError(t, err)
	assert.Assert(t, rvi != nil)

}
