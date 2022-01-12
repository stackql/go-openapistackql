package openapistackql_test

import (
	. "openapistackql"
	"testing"

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

	outFile, err := GetFilePathFromRepositoryRoot("test/output/Application.spew.txt")

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

	s := svc.AsSourceString()

	assert.Assert(t, s != "")

	t.Logf("TestSimpleOktaApplicationServiceReadAndDump passed")
}
