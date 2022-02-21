package openapistackql_test

import (
	"encoding/json"
	"testing"

	. "github.com/stackql/go-openapistackql/openapistackql"

	"gotest.tools/assert"
)

const (
	individualDownloadAllowedRegistryCfgStr string = `{"allowSrcDownload": true, "useEmbedded": false}`
	pullProvidersRegistryCfgStr             string = `{"srcPrefix": "test-src", "useEmbedded": false}`
)

func TestRegistrySimpleOktaApplicationServiceRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistrySimpleOktaApplicationServiceRead)
}

func TestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistryIndirectGoogleComputeResourcesJsonRead)
}

func TestRegistryIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistryIndirectGoogleComputeServiceSubsetJsonRead)
}

func TestRegistryIndirectGoogleComputeServiceSubsetAccess(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistryIndirectGoogleComputeServiceSubsetAccess)
}

func TestLocalRegistryIndirectGoogleComputeServiceSubsetAccess(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistryIndirectGoogleComputeServiceSubsetAccess)
}

func TestProviderPull(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, pullProvidersRegistryCfgStr, execTestRegistrySimpleOktaPull)
}

func TestProviderPullAndPersist(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, pullProvidersRegistryCfgStr, execTestRegistrySimpleOktaPullAndPersist)
}

func execLocalAndRemoteRegistryTests(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	var rc RegistryConfig
	if registryConfigStr != "" {
		err := json.Unmarshal([]byte(registryConfigStr), &rc)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
	}

	r, err := GetMockRegistry(rc)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	tf(t, r)

	r, err = GetMockLocalRegistry(rc)
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

func execTestRegistrySimpleOktaPull(t *testing.T, r RegistryAPI) {
	arc, err := r.PullProviderArchive("okta", "v1")

	assert.NilError(t, err)

	assert.Assert(t, arc != nil)

}

func execTestRegistrySimpleOktaPullAndPersist(t *testing.T, r RegistryAPI) {
	err := r.PullAndPersistProviderArchive("okta", "v1")

	assert.NilError(t, err)

	svc, err := r.GetService("okta/v1/services/Application.yaml")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	assert.Equal(t, svc.GetName(), "application")

	t.Logf("TestRegistrySimpleOktaPullAndPersist passed")

}
