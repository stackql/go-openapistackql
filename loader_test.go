package openapistackql_test

import (
	"os"
	"testing"

	. "github.com/stackql/openapistackql"

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

	svc, err := LoadServiceDocFromBytes(b)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	outFile, err := GetFilePathFromRepositoryRoot("test/_output/Application.spew.raw.txt")

	assert.NilError(t, err)

	err = svc.ToSourceFile(outFile)

	assert.NilError(t, err)

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed")
}

func TestSimpleOktaApplicationServiceReadAndDumpString(t *testing.T) {
	b, err := GetServiceDocBytes("okta/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	svc, err := LoadServiceDocFromBytes(b)
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

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed")
}
