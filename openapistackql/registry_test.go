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
	awsTestableVersions = []string{
		"v0.1.0",
	}
	oktaTestableVersions = []string{
		"v0.1.0",
	}
	googleTestableVersions = []string{
		// "v0.1.0",
		"v0.1.2",
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

func TestRegistryCanHandleUnspecifiedResponseWithDefaults(t *testing.T) {
	execLocalRegistryTestOnly(t, unsignedProvidersRegistryCfgStr, execTestRegistryCanHandleUnspecifiedResponseWithDefaults)
}

func TestRegistryCanHandlePolymorphismAllOf(t *testing.T) {
	execLocalRegistryTestOnly(t, unsignedProvidersRegistryCfgStr, execTestRegistryCanHandlePolymorphismAllOf)
}

func TestListProvidersRegistry(t *testing.T) {
	execRemoteRegistryTestOnly(t, unsignedProvidersRegistryCfgStr, execTestRegistryProvidersList)
}

func TestListProviderVersionsRegistry(t *testing.T) {
	execRemoteRegistryTestOnly(t, unsignedProvidersRegistryCfgStr, execTestRegistryProviderVersionsList)
}

func execLocalAndRemoteRegistryTests(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	rc, err := getRegistryCfgFromString(registryConfigStr)

	assert.NilError(t, err)

	runRemote(t, rc, tf)

	runLocal(t, rc, tf)
}

func execLocalRegistryTestOnly(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	rc, err := getRegistryCfgFromString(registryConfigStr)

	assert.NilError(t, err)

	runLocal(t, rc, tf)
}

func execRemoteRegistryTestOnly(t *testing.T, registryConfigStr string, tf func(t *testing.T, r RegistryAPI)) {

	rc, err := getRegistryCfgFromString(registryConfigStr)

	assert.NilError(t, err)

	runRemote(t, rc, tf)
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
	for _, vr := range oktaTestableVersions {
		pr, err := LoadProviderByName("okta", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		ps, ok := pr.ProviderServices["application"]
		if !ok {
			t.Fatalf("Test failed: could not locate ProviderService for okta.application")
		}
		svc, err := r.GetService(ps)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Equal(t, svc.GetName(), "application")
	}

	t.Logf("TestSimpleOktaServiceRead passed")
}

func execTestRegistryProvidersList(t *testing.T, r RegistryAPI) {

	pr, err := r.ListAllAvailableProviders()
	assert.NilError(t, err)

	assert.Assert(t, len(pr) > 0)
	assert.Assert(t, len(pr["google"].Versions) == 1)
	assert.Assert(t, len(pr["okta"].Versions) == 1)
	assert.Assert(t, pr["google"].Versions[0] == "v2.0.1")
	assert.Assert(t, pr["okta"].Versions[0] == "v2.0.1")

	t.Logf("execTestRegistryProvidersList passed")
}

func execTestRegistryProviderVersionsList(t *testing.T, r RegistryAPI) {

	pr, err := r.ListAllProviderVersions("google")
	assert.NilError(t, err)

	assert.Assert(t, len(pr) == 1)
	assert.Assert(t, len(pr["google"].Versions) == 2)

	t.Logf("execTestRegistryProviderVersionsList passed")
}

func execTestRegistryIndirectGoogleComputeResourcesJsonRead(t *testing.T, r RegistryAPI) {

	for _, vr := range googleTestableVersions {
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

	for _, vr := range googleTestableVersions {
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

		sv, err := r.GetService(rr.Resources["acceleratorTypes"].Methods["get"].ProviderService)

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

	for _, vr := range googleTestableVersions {
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

	for _, vr := range oktaTestableVersions {
		arc, err := r.PullProviderArchive("okta", vr)

		assert.NilError(t, err)

		assert.Assert(t, arc != nil)
	}

}

func execTestRegistrySimpleOktaPullAndPersist(t *testing.T, r RegistryAPI) {
	for _, vr := range oktaTestableVersions {
		err := r.PullAndPersistProviderArchive("okta", vr)

		assert.NilError(t, err)

		pr, err := LoadProviderByName("okta", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		ps, ok := pr.ProviderServices["application"]
		if !ok {
			t.Fatalf("Test failed: could not locate ProviderService for okta.application")
		}
		svc, err := r.GetService(ps)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Equal(t, svc.GetName(), "application")
	}

	t.Logf("TestRegistrySimpleOktaPullAndPersist passed")

}

func execTestRegistryIndirectGoogleComputeServiceMethodResolutionSeparateDocs(t *testing.T, r RegistryAPI) {

	for _, vr := range googleTestableVersions {
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

		os, remainingParams, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", matchParams)

		assert.Assert(t, ok)

		assert.Assert(t, len(remainingParams) == 0)

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

		os, remainingParams, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", matchParams)

		assert.Assert(t, ok)

		assert.Assert(t, len(remainingParams) == 0)

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

}

func execTestRegistryCanHandleUnspecifiedResponseWithDefaults(t *testing.T, r RegistryAPI) {

	for _, vr := range []string{"v0.1.2"} {
		pr, err := r.LoadProviderByName("google", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		sh, err := pr.GetProviderService("compute")

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Assert(t, sh != nil)

		sv, err := r.GetServiceFragment(sh, "disks")

		assert.NilError(t, err)

		assert.Assert(t, sv != nil)

		sn := sv.GetName()

		assert.Equal(t, sn, "compute")

		rsc, err := sv.GetResource("disks")

		assert.NilError(t, err)

		matchParams := map[string]interface{}{
			"project": struct{}{},
			"zone":    struct{}{},
		}

		os, remainingParams, ok := rsc.GetFirstMethodMatchFromSQLVerb("select", matchParams)

		assert.Assert(t, ok)

		assert.Assert(t, len(remainingParams) == 0)

		assert.Equal(t, os.OperationRef.Value.OperationID, "compute.disks.list")

		sc, _, err := os.GetResponseBodySchemaAndMediaType()

		assert.NilError(t, err)

		assert.Equal(t, sc.Type, "object")

		items, _ := sc.GetSelectListItems("items")

		assert.Assert(t, items != nil)

		name, nameExists := items.Items.Value.Properties["name"]

		assert.Assert(t, nameExists)

		assert.Equal(t, name.Value.Type, "string")

		id, idExists := items.Items.Value.Properties["id"]

		assert.Assert(t, idExists)

		assert.Equal(t, id.Value.Type, "string")
	}

}

func execTestRegistryCanHandlePolymorphismAllOf(t *testing.T, r RegistryAPI) {

	for _, vr := range []string{"v1"} {
		pr, err := r.LoadProviderByName("github", vr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		sh, err := pr.GetProviderService("apps")

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		assert.Assert(t, sh != nil)

		sv, err := r.GetServiceFragment(sh, "apps")

		assert.NilError(t, err)

		assert.Assert(t, sv != nil)

		// sn := sv.GetName()

		// assert.Equal(t, sn, "repos")

		rsc, err := sv.GetResource("apps")

		assert.NilError(t, err)

		os, ok := rsc.Methods.FindMethod("create_from_manifest")

		assert.Assert(t, ok)

		assert.Equal(t, os.OperationRef.Value.OperationID, "apps/create-from-manifest")

		assert.Equal(t, os.OperationRef.Value.Responses["201"].Value.Content["application/json"].Schema.Value.Type, "")

		sVal := NewSchema(os.OperationRef.Value.Responses["201"].Value.Content["application/json"].Schema.Value, sv, "", os.OperationRef.Value.Responses["201"].Value.Content["application/json"].Schema.Ref)

		tab := sVal.Tabulate(false)

		colz := tab.GetColumns()

		for _, expectedProperty := range []string{"pem", "description"} {
			found := false
			for _, col := range colz {
				if col.Name == expectedProperty {
					found = true
					break
				}
			}
			assert.Assert(t, found)
		}
	}

}

func TestRegistryProviderLatestVersion(t *testing.T) {

	rc, err := getRegistryCfgFromString(individualDownloadAllowedRegistryCfgStr)
	assert.NilError(t, err)
	r, err := GetMockLocalRegistry(rc)
	assert.NilError(t, err)
	v, err := r.GetLatestAvailableVersion("google")
	assert.NilError(t, err)
	assert.Equal(t, v, "v0.1.2")
	vo, err := r.GetLatestAvailableVersion("okta")
	assert.NilError(t, err)
	assert.Equal(t, vo, "v0.1.0")

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
