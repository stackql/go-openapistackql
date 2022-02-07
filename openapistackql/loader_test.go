package openapistackql_test

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/stackql/go-openapistackql/openapistackql"

	"gotest.tools/assert"
)

func TestSimpleOktaApplicationServiceRead(t *testing.T) {
	b, err := GetServiceDocBytes("okta/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	svc, err := LoadServiceDocFromBytes(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	t.Logf("TestSimpleOktaServiceRead passed")
}

func TestSimpleOktaApplicationServiceReadAndDump(t *testing.T) {
	b, err := GetServiceDocBytes("okta/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	outFile, err := GetFilePathFromRepositoryRoot("test/_output/Application.spew.raw.txt")

	assert.NilError(t, err)

	err = svc.ToSourceFile(outFile)

	assert.NilError(t, err)

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed\n")
}

func TestSimpleOktaApplicationServiceReadAndDumpString(t *testing.T) {
	b, err := GetServiceDocBytes("okta/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	outFile, err := GetFilePathFromRepositoryRoot("test/_output/Application.spew.go")

	assert.NilError(t, err)

	s := svc.AsSourceString()

	assert.Assert(t, s != "")

	f, err := os.OpenFile(outFile, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0666)
	assert.NilError(t, err)

	f.WriteString("package main\n\n")
	f.WriteString("import(\n")
	f.WriteString(`  "encoding/json"` + "\n\n")
	f.WriteString(`  "github.com/getkin/kin-openapi/openapi3"` + "\n")
	f.WriteString(`  "github.com/stackql/openapistackql"` + "\n")
	f.WriteString(")\n\n")
	f.WriteString("var Svc *openapistackql.Service = " + s)

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed\n")
}

func TestSimpleOktaApplicationServiceJsonReadAndDumpString(t *testing.T) {
	b, err := GetServiceDocBytes("okta/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	outFile, err := GetFilePathFromRepositoryRoot("test/_output/Application.json")

	assert.NilError(t, err)

	b, err = json.MarshalIndent(svc, "", "  ")

	assert.NilError(t, err)

	assert.Assert(t, b != nil)

	f, err := os.OpenFile(outFile, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0666)
	assert.NilError(t, err)

	f.Write(b)
	f.Close()

	// ob, err := os.ReadFile(outFile)
	// assert.NilError(t, err)

	// var sv Service
	// err = json.Unmarshal(ob, &sv)
	// assert.NilError(t, err)

	// assert.Assert(t, sv.Components.Schemas != nil)

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed\n")
}

func TestSimpleGoogleComputeServiceJsonReadAndDumpString(t *testing.T) {
	b, err := GetServiceDocBytes("googleapis.com/services-split/compute/compute-v1.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	l := NewLoader()

	svc, err := l.LoadFromBytes(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "compute")

	outFile, err := GetFilePathFromRepositoryRoot("test/_output/Compute.json")

	assert.NilError(t, err)

	b, err = json.MarshalIndent(svc, "", "  ")

	assert.NilError(t, err)

	assert.Assert(t, b != nil)

	f, err := os.OpenFile(outFile, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0666)
	assert.NilError(t, err)

	f.Write(b)
	f.Close()

	// ob, err := os.ReadFile(outFile)
	// assert.NilError(t, err)

	// var sv Service
	// err = json.Unmarshal(ob, &sv)
	// assert.NilError(t, err)

	// assert.Assert(t, sv.Components.Schemas != nil)

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed\n")
}

func TestSimpleGoogleComputeResourcesJsonRead(t *testing.T) {
	b, err := GetServiceDocBytes("googleapis.com/resources/compute-v1.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	rr, err := LoadResourcesShallow(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, rr != nil)
	assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")
	assert.Equal(t, rr.ServiceDocPath.Ref, "googleapis.com/services-split/compute/compute-v1.yaml")

	t.Logf("TestSimpleGoogleComputeResourcesJsonRead passed\n")
}

func TestIndirectGoogleComputeResourcesJsonRead(t *testing.T) {

	pr, err := LoadProviderByName("google")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	rr, err := pr.GetResourcesShallow("compute")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, rr != nil)
	assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")
	assert.Equal(t, rr.ServiceDocPath.Ref, "googleapis.com/services-split/compute/compute-v1.yaml")

	t.Logf("TestSimpleGoogleComputeResourcesJsonRead passed\n")
}

func TestIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T) {

	pr, err := LoadProviderByName("google")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	rr, err := pr.GetResourcesShallow("compute")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, rr != nil)
	assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")
	assert.Equal(t, rr.ServiceDocPath.Ref, "googleapis.com/services-split/compute/compute-v1.yaml")

	sb, err := GetServiceDocBytes(rr.ServiceDocPath.Ref)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	sv, err := LoadServiceSubsetDocFromBytes(rr, "instances", sb)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	assert.Assert(t, sv != nil)

	sn := sv.GetName()

	assert.Equal(t, sn, "compute")

	t.Logf("TestIndirectGoogleComputeServiceSubsetJsonRead passed\n")
}

func TestIndirectGoogleComputeServiceSubsetAccess(t *testing.T) {

	pr, err := LoadProviderByName("google")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	sh, err := pr.GetProviderService("compute")

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, sh != nil)

	sv, err := sh.GetServiceFragment("instances")

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, sv != nil)

	sn := sv.GetName()

	assert.Equal(t, sn, "compute")

	t.Logf("TestIndirectGoogleComputeServiceSubsetAccess passed\n")
}
