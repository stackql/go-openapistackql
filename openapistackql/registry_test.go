package openapistackql_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	. "github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/go-openapistackql/pkg/fileutil"

	"gotest.tools/assert"
)

var (
	testableVersions = []string{
		"v1",
		"v2.0.1",
	}
)

const (
	individualDownloadAllowedRegistryCfgStr string = `{"allowSrcDownload": true }`
	pullProvidersRegistryCfgStr             string = `{"srcPrefix": "test-src" }`
	deprecatedRegistryCfgStr                string = `{"srcPrefix": "deprecated-src" }`
	unsignedProvidersRegistryCfgStr         string = `{"srcPrefix": "unsigned-src",  "verifyConfig": { "nopVerify": true }  }`
)

func init() {
	var err error
	OpenapiFileRoot, err = fileutil.GetFilePathFromRepositoryRoot("providers")
	if err != nil {
		os.Exit(1)
	}
}

func TestRegistrySimpleOktaApplicationServiceRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistrySimpleOktaApplicationServiceRead)
}

func TestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T) {
	execLocalAndRemoteRegistryTests(t, individualDownloadAllowedRegistryCfgStr, execTestRegistryIndirectGoogleComputeResourcesJsonRead)
}

func TestRegistryIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T) {
	execLocalAndRemoteRegistryTestsIndividualDownloadAllowed(t, individualDownloadAllowedRegistryCfgStr, execTestRegistryIndirectGoogleComputeServiceSubsetJsonRead)
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

func TestRegistryIndirectGoogleComputeServiceMethodResolutionSeparateDocs(t *testing.T) {
	execLocalRegistryTestOnly(t, unsignedProvidersRegistryCfgStr, execTestRegistryIndirectGoogleComputeServiceMethodResolutionSeparateDocs)
}

func TestRegistryArrayTopLevelResponse(t *testing.T) {
	execLocalRegistryTestOnly(t, unsignedProvidersRegistryCfgStr, execTestRegistryCanHandleArrayResponts)
}

func execLocalAndRemoteRegistryTests(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	rc, err := getRegistryCfgFromString(registryConfigStr)

	assert.NilError(t, err)

	runRemote(t, rc, tf)

	runLocal(t, rc, tf)
}

func execLocalAndRemoteRegistryTestsIndividualDownloadAllowed(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	rc, err := getRegistryCfgFromString(registryConfigStr)

	rc.AllowSrcDownload = true

	assert.NilError(t, err)

	runRemote(t, rc, tf)

	runLocal(t, rc, tf)
}

func execLocalRegistryTestOnly(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	rc, err := getRegistryCfgFromString(registryConfigStr)

	assert.NilError(t, err)

	runLocal(t, rc, tf)
}

func getRegistryCfgFromString(registryConfigStr string) (RegistryConfig, error) {
	var rc RegistryConfig
	if registryConfigStr != "" {
		err := json.Unmarshal([]byte(registryConfigStr), &rc)
		return rc, err
	}
	return rc, fmt.Errorf("could not compose registry config")
}

func runLocal(t *testing.T, rc RegistryConfig, tf func(t *testing.T, r RegistryAPI)) {
	r, err := GetMockLocalRegistry(rc)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	tf(t, r)
}

func runRemote(t *testing.T, rc RegistryConfig, tf func(t *testing.T, r RegistryAPI)) {
	r, err := GetMockRegistry(rc)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	tf(t, r)
}

func execTestRegistrySimpleOktaApplicationServiceRead(t *testing.T, r RegistryAPI) {
	for _, vr := range testableVersions {
		svc, err := r.GetService(fmt.Sprintf("okta/%s/services/Application.yaml", vr))
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Equal(t, svc.GetName(), "application")
	}

	t.Logf("TestSimpleOktaServiceRead passed")
}

func execTestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T, r RegistryAPI) {

	for _, vr := range testableVersions {
		pr, err := r.LoadProviderByName("google", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		rr, err := r.GetResourcesShallowFromProvider(pr, "compute")
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Assert(t, rr != nil)
		assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")
	}
	t.Logf("TestSimpleGoogleComputeResourcesJsonRead passed\n")
}

func execTestRegistryIndirectGoogleComputeServiceSubsetJsonRead(t *testing.T, r RegistryAPI) {

	for _, vr := range testableVersions {
		pr, err := r.LoadProviderByName("google", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		rr, err := r.GetResourcesShallowFromProvider(pr, "compute")
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Assert(t, rr != nil)
		assert.Equal(t, rr.Resources["acceleratorTypes"].ID, "google.compute.acceleratorTypes")

		sv, err := r.GetService(rr.Resources["acceleratorTypes"].Methods["get"].OperationRef.ExtractServiceDocPath())

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		assert.Assert(t, sv != nil)

		sn := sv.GetName()

		assert.Equal(t, sn, "compute")
	}

	t.Logf("TestIndirectGoogleComputeServiceSubsetJsonRead passed\n")
}

func execTestRegistryIndirectGoogleComputeServiceSubsetAccess(t *testing.T, r RegistryAPI) {

	for _, vr := range testableVersions {
		pr, err := r.LoadProviderByName("google", vr)
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
	}

	t.Logf("TestIndirectGoogleComputeServiceSubsetAccess passed\n")
}

func execTestRegistrySimpleOktaPull(t *testing.T, r RegistryAPI) {

	for _, vr := range testableVersions {
		arc, err := r.PullProviderArchive("okta", vr)

		assert.NilError(t, err)

		assert.Assert(t, arc != nil)
	}

}

func execTestRegistrySimpleOktaPullAndPersist(t *testing.T, r RegistryAPI) {
	for _, vr := range testableVersions {
		err := r.PullAndPersistProviderArchive("okta", vr)

		assert.NilError(t, err)

		svc, err := r.GetService(fmt.Sprintf("okta/%s/services/Application.yaml", vr))
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Equal(t, svc.GetName(), "application")
	}

	t.Logf("TestRegistrySimpleOktaPullAndPersist passed")

}

func execTestRegistryIndirectGoogleComputeServiceMethodResolutionSeparateDocs(t *testing.T, r RegistryAPI) {

	for _, vr := range testableVersions {
		pr, err := r.LoadProviderByName("google", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		sh, err := pr.GetProviderService("compute")

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Assert(t, sh != nil)

		sv, err := r.GetServiceFragment(sh, "acceleratorTypes")

		assert.NilError(t, err)

		assert.Assert(t, sv != nil)

		sn := sv.GetName()

		assert.Equal(t, sn, "compute")

		rsc, err := sv.GetResource("acceleratorTypes")

		assert.NilError(t, err)

		matchParams := map[string]interface{}{
			"project": struct{}{},
		}

		os, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", matchParams)

		assert.Assert(t, ok)

		assert.Equal(t, os.OperationRef.Value.OperationID, "compute.acceleratorTypes.aggregatedList")
	}

	t.Logf("TestRegistryIndirectGoogleComputeServiceMethodResolutionSeparateDocs passed\n")
}

func execTestRegistryCanHandleArrayResponts(t *testing.T, r RegistryAPI) {

	for _, vr := range []string{"v1"} {
		pr, err := r.LoadProviderByName("github", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		sh, err := pr.GetProviderService("repos")

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Assert(t, sh != nil)

		sv, err := r.GetServiceFragment(sh, "repos")

		assert.NilError(t, err)

		assert.Assert(t, sv != nil)

		// sn := sv.GetName()

		// assert.Equal(t, sn, "repos")

		rsc, err := sv.GetResource("repos")

		assert.NilError(t, err)

		matchParams := map[string]interface{}{
			"org": struct{}{},
		}

		os, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", matchParams)

		assert.Assert(t, ok)

		assert.Equal(t, os.OperationRef.Value.OperationID, "repos/list-for-org")

		assert.Equal(t, os.OperationRef.Value.Responses["200"].Value.Content["application/json"].Schema.Value.Type, "array")

		props := os.OperationRef.Value.Responses["200"].Value.Content["application/json"].Schema.Value.Items.Value.Properties

		name, nameExists := props["name"]

		assert.Assert(t, nameExists)

		assert.Equal(t, name.Value.Type, "string")

		sshUrl, sshUrlExists := props["ssh_url"]

		assert.Assert(t, sshUrlExists)

		assert.Equal(t, sshUrl.Value.Type, "string")
	}

	t.Logf("TestRegistryIndirectGoogleComputeServiceMethodResolutionSeparateDocs passed\n")
}

func TestRegistryProviderLatestVersion(t *testing.T) {

	rc, err := getRegistryCfgFromString(individualDownloadAllowedRegistryCfgStr)
	assert.NilError(t, err)
	r, err := GetMockLocalRegistry(rc)
	assert.NilError(t, err)
	v, err := r.GetLatestAvailableVersion("google")
	assert.NilError(t, err)
	assert.Equal(t, v, "v2.0.1")
	vo, err := r.GetLatestAvailableVersion("okta")
	assert.NilError(t, err)
	assert.Equal(t, vo, "v2.0.1")

	rc, err = getRegistryCfgFromString(deprecatedRegistryCfgStr)
	assert.NilError(t, err)
	r, err = GetMockLocalRegistry(rc)
	assert.NilError(t, err)
	v, err = r.GetLatestAvailableVersion("google")
	assert.NilError(t, err)
	assert.Equal(t, v, "v1")
	vo, err = r.GetLatestAvailableVersion("okta")
	assert.NilError(t, err)
	assert.Equal(t, vo, "v1")

	t.Logf("TestRegistryProviderLatestVersion passed\n")
}
