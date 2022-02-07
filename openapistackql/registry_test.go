package openapistackql_test

import (
	"testing"

	. "github.com/stackql/go-openapistackql/openapistackql"

	"gotest.tools/assert"
)

func TestRegistrySimpleOktaApplicationServiceRead(t *testing.T) {
	r, err := GetMockRegistry()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	svc, err := r.GetService("okta/v1/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	t.Logf("TestSimpleOktaServiceRead passed")
}
