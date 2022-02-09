package openapistackql_test

import (
	"testing"

	. "github.com/stackql/go-openapistackql/openapistackql"

	"gotest.tools/assert"
)

func TestRegistrySimpleOktaApplicationServiceRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, execTestRegistrySimpleOktaApplicationServiceRead)
}

func TestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, execTestRegistryIndirectGoogleComputeResourcesJsonRead)
}

func TestRegistryIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, execTestRegistryIndirectGoogleComputeServiceSubsetJsonRead)
}

func TestRegistryIndirectGoogleComputeServiceSubsetAccess(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, execTestRegistryIndirectGoogleComputeServiceSubsetAccess)
}

func TestLocalRegistryIndirectGoogleComputeServiceSubsetAccess(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, execTestRegistryIndirectGoogleComputeServiceSubsetAccess)
}

func execLocalAndRemoteRegistryTests(t *testing.T, tf func(t *testing.T, r RegistryAPI)) {

	r, err := GetMockRegistry()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	tf(t, r)

	r, err = GetMockLocalRegistry()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	tf(t, r)
}

func execTestRegistrySimpleOktaApplicationServiceRead(t *testing.T, r RegistryAPI) {
	svc, err := r.GetService("okta/v1/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	t.Logf("TestSimpleOktaServiceRead passed")
}

func execTestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T, r RegistryAPI) {

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

func execTestRegistryIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T, r RegistryAPI) {

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

func execTestRegistryIndirectGoogleComputeServiceSubsetAccess(t *testing.T, r RegistryAPI) {

	pr, err := r.LoadProviderByName("google", "v1")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	sh, err := pr.GetProviderService("compute")

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, sh != nil)

	sv, err := r.GetServiceFragment(sh, "instances")

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Assert(t, sv != nil)

	sn := sv.GetName()

	assert.Equal(t, sn, "compute")

	t.Logf("TestIndirectGoogleComputeServiceSubsetAccess passed\n")
}
