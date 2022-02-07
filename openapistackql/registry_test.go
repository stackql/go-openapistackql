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

func TestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T) {

	r, err := GetMockRegistry()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	pr, err := r.LoadProviderByName("google", "v1")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	rr, err := r.GetResourcesShallowFromProvider(pr, "compute")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, rr != nil)
	assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")
	assert.Equal(t, rr.ServiceDocPath.Ref, "googleapis.com/v1/services-split/compute/compute-v1.yaml")

	t.Logf("TestSimpleGoogleComputeResourcesJsonRead passed\n")
}

func TestRegistryIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T) {

	r, err := GetMockRegistry()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	pr, err := r.LoadProviderByName("google", "v1")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	rr, err := r.GetResourcesShallowFromProvider(pr, "compute")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, rr != nil)
	assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")
	assert.Equal(t, rr.ServiceDocPath.Ref, "googleapis.com/v1/services-split/compute/compute-v1.yaml")

	sv, err := r.GetService(rr.ServiceDocPath.Ref)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	assert.Assert(t, sv != nil)

	sn := sv.GetName()

	assert.Equal(t, sn, "compute")

	t.Logf("TestIndirectGoogleComputeServiceSubsetJsonRead passed\n")
}
