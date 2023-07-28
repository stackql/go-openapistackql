package openapistackql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	yamlconv "github.com/ghodss/yaml"
	"github.com/go-openapi/jsonpointer"
	yaml "gopkg.in/yaml.v2"
)

const (
	ConfigFilesMode fs.FileMode = 0664
)

var (
	IgnoreEmbedded  bool
	OpenapiFileRoot string
	_               Loader = &standardLoader{}
)

func init() {
	OpenapiFileRoot = "."
}

type DiscoveryDoc interface {
	iDiscoveryDoc()
}

type Loader interface {
	LoadFromBytes(bytes []byte) (Service, error)
	LoadFromBytesAndResources(rr ResourceRegister, resourceKey string, bytes []byte) (Service, error)
	//
	extractAndMergeQueryTransposeServiceLevel(svc Service) error
}

type standardLoader struct {
	*openapi3.Loader
	//
	visitedExpectedRequest  map[Schema]struct{}
	visitedExpectedResponse map[Schema]struct{}
	visitedOperation        map[*openapi3.Operation]struct{}
	visitedOperationStore   map[OperationStore]struct{}
	visitedPathItem         map[*openapi3.PathItem]struct{}
}

func LoadResourcesShallow(ps ProviderService, bt []byte) (ResourceRegister, error) {
	return loadResourcesShallow(ps, bt)
}

func loadResourcesShallow(ps ProviderService, bt []byte) (ResourceRegister, error) {
	rv := newStandardResourceRegister()
	err := yaml.Unmarshal(bt, &rv)
	if err != nil {
		return nil, err
	}
	p, provExists := ps.GetProvider()
	if !provExists {
		return nil, errors.New("provider not found")
	}
	rv.SetProvider(p)
	rv.SetProviderService(ps)
	resourceregisterLoadBackwardsCompatibility(rv)
	return rv, nil
}

func (l *standardLoader) LoadFromBytes(bytes []byte) (Service, error) {
	doc, err := l.LoadFromData(bytes)
	if err != nil {
		return nil, err
	}
	svc := NewService(doc)
	err = l.extractResources(svc)
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (l *standardLoader) LoadFromBytesAndResources(rr ResourceRegister, resourceKey string, bytes []byte) (Service, error) {
	doc, err := l.LoadFromData(bytes)
	if err != nil {
		return nil, err
	}
	svc := NewService(doc)
	docUrl := rr.ObtainServiceDocUrl(resourceKey)
	if docUrl != "" {
		err = l.mergeResourcesScoped(svc, docUrl, rr)
	} else {
		err = l.mergeResources(svc, rr.GetResources(), rr.GetServiceDocPath())
	}
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (l *standardLoader) extractResources(svc Service) error {
	rscs, ok := svc.GetComponents().Extensions[ExtensionKeyResources]
	if !ok {
		return fmt.Errorf("Service.extractResources() failure")
	}
	var bt []byte
	var err error
	switch rs := rscs.(type) {
	case json.RawMessage:
		bt, err = rs.MarshalJSON()
	default:
		bt, err = yaml.Marshal(rscs)
	}
	if err != nil {
		return err
	}
	rscMap := make(map[string]*standardResource)
	err = yaml.Unmarshal(bt, rscMap)
	castMap := make(map[string]Resource, len(rscMap))
	for k, v := range rscMap {
		castMap[k] = v
	}
	if err != nil {
		return err
	}
	return l.mergeResources(svc, castMap, nil)
}

func (l *standardLoader) extractAndMergeGraphQL(operation OperationStore) error {
	if operation.GetOperationRef() == nil || operation.GetOperationRef().Value == nil {
		return nil
	}
	gql, ok := operation.GetOperationRef().Value.Extensions[ExtensionKeyGraphQL]
	if !ok {
		return nil
	}
	var bt []byte
	var err error
	switch rs := gql.(type) {
	case json.RawMessage:
		bt, err = rs.MarshalJSON()
	default:
		bt, err = yaml.Marshal(gql)
	}
	if err != nil {
		return err
	}
	var rv standardGraphQL
	err = yaml.Unmarshal(bt, &rv)
	if err != nil {
		return err
	}
	operation.setGraphQL(&rv)
	return nil
}

func extractStackQLConfig(qt interface{}) (StackQLConfig, error) {
	var bt []byte
	var err error
	switch rs := qt.(type) {
	case json.RawMessage:
		bt, err = rs.MarshalJSON()
	default:
		bt, err = yaml.Marshal(qt)
	}
	if err != nil {
		return nil, err
	}
	var rv standardStackQLConfig
	err = yaml.Unmarshal(bt, &rv)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

func (l *standardLoader) extractAndMergeQueryTransposeOpLevel(operation OperationStore) error {
	if operation.GetOperationRef() == nil || operation.GetOperationRef().Value == nil {
		return nil
	}
	qt, ok := operation.GetOperationRef().Value.Extensions[ExtensionKeyConfig]
	if !ok {
		return nil
	}
	rv, err := extractStackQLConfig(qt)
	if err != nil {
		return err
	}
	operation.setStackQLConfig(rv)
	return nil
}

func (l *standardLoader) extractAndMergeQueryTransposeServiceLevel(svc Service) error {
	qt, ok := svc.getExtension(ExtensionKeyConfig)
	if !ok {
		return nil
	}
	rv, err := extractStackQLConfig(qt)
	if err != nil {
		return err
	}
	svc.setStackQLConfig(rv)
	return nil
}

func (l *standardLoader) extractAndMergeConfigServiceLevel(svc Service) error {
	qt, ok := svc.getExtension(ExtensionKeyConfig)
	if !ok {
		return nil
	}
	rv, err := extractStackQLConfig(qt)
	if err != nil {
		return err
	}
	svc.setStackQLConfig(rv)
	return nil
}

func (l *standardLoader) mergeResources(svc Service, rscMap map[string]Resource, sdRef *ServiceRef) error {
	rscCast := make(map[string]*standardResource, len(rscMap))
	for k, rsc := range rscMap {
		rscCast[k] = rsc.(*standardResource)
		var sr *ServiceRef
		if sdRef != nil {
			sr = sdRef
		}
		if rsc.GetServiceDocPath() != nil {
			sr = rsc.GetServiceDocPath()
		}
		err := l.mergeResource(svc, rsc, sr)
		if err != nil {
			return err
		}
	}
	svc.setResourceMap(rscCast)
	return nil
}

func (l *standardLoader) mergeResourcesScoped(svc Service, svcUrl string, rr ResourceRegister) error {
	scopedMap := make(map[string]Resource)
	for k, rsc := range rr.GetResources() {
		if rr.ObtainServiceDocUrl(k) == svcUrl {
			err := l.mergeResource(svc, rsc, &ServiceRef{Ref: svcUrl})
			if err != nil {
				return err
			}
			scopedMap[k] = rsc
		}
	}
	rsc, _ := svc.GetResources()
	scopedCast := make(map[string]*standardResource, len(scopedMap))
	for k, v := range scopedMap {
		scopedCast[k] = v.(*standardResource)
	}
	if len(rsc) == 0 {
		svc.setResourceMap(scopedCast)
		return nil
	}
	return nil
}

func (l *standardLoader) mergeResource(svc Service,
	rsc Resource,
	sr *ServiceRef,
) error {
	rsc.setService(svc) // must happen before resolving inverses
	for k, vOp := range rsc.GetMethods() {
		v := vOp
		v.setMethodKey(k)
		err := l.resolveOperationRef(svc, rsc, &v, v.GetPathRef(), sr)
		if err != nil {
			return err
		}
		req, reqExists := v.GetRequest()
		if !reqExists && v.GetOperationRef().Value.RequestBody != nil {
			req = &standardExpectedRequest{}
			v.setRequest(req.(*standardExpectedRequest))
		}
		err = l.resolveExpectedRequest(svc, v.GetOperationRef().Value, req)
		if err != nil {
			return err
		}
		response, responseExists := v.GetResponse()
		if !responseExists && v.GetOperationRef().Value.Responses != nil {
			response = &standardExpectedResponse{}
			v.setResponse(response.(*standardExpectedResponse))
		}
		err = l.resolveExpectedResponse(svc, v.GetOperationRef().Value, response)
		if err != nil {
			return err
		}
		iv := openapi3.Servers(svc.GetServers())
		v.setServers(&iv)
		rsc.setMethod(k, &v)
	}
	for sqlVerb, dir := range rsc.getSQLVerbs() {
		for i, v := range dir {
			cur := v
			err := l.resolveSQLVerb(rsc, &cur, sqlVerb)
			if err != nil {
				return err
			}
			rsc.mutateSQLVerb(sqlVerb, i, cur)
		}
	}
	// TODO: add second pass for inverse ops
	for sqlVerb, dir := range rsc.getSQLVerbs() {
		for i, v := range dir {
			cur := v
			_, err := latePassResolveInverse(svc, &cur)
			if err != nil {
				return err
			}
			rsc.mutateSQLVerb(sqlVerb, i, cur)
		}
	}
	rsc.setProvider(svc.getProvider())
	rsc.setProviderService(svc.getProviderService())
	return nil
}

func (svc *standardService) ToJson() ([]byte, error) {
	return svc.MarshalJSON()
}

func (svc *standardService) ToYaml() ([]byte, error) {
	j, err := svc.ToJson()
	if err != nil {
		return nil, err
	}
	return yamlconv.JSONToYAML(j)
}

func (pr *standardProvider) ToJson() ([]byte, error) {
	return pr.MarshalJSON()
}

func (pr *standardProvider) ToYaml() ([]byte, error) {
	j, err := pr.ToJson()
	if err != nil {
		return nil, err
	}
	return yamlconv.JSONToYAML(j)
}

func (svc *standardService) ToYamlFile(filePath string) error {
	bytes, err := svc.ToYaml()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, ConfigFilesMode)
}

func (pr *standardProvider) ToYamlFile(filePath string) error {
	bytes, err := pr.ToYaml()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, ConfigFilesMode)
}

func NewLoader() Loader {
	return &standardLoader{
		&openapi3.Loader{Context: context.Background()},
		make(map[Schema]struct{}),
		make(map[Schema]struct{}),
		make(map[*openapi3.Operation]struct{}),
		make(map[OperationStore]struct{}),
		make(map[*openapi3.PathItem]struct{}),
	}
}

func LoadServiceDocFromBytes(ps ProviderService, bytes []byte) (Service, error) {
	return loadServiceDocFromBytes(ps, bytes)
}

func LoadProviderDocFromBytes(bytes []byte) (Provider, error) {
	return loadProviderDocFromBytes(bytes)
}

func LoadServiceDocFromFile(ps ProviderService, fileName string) (Service, error) {
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return loadServiceDocFromBytes(ps, bytes)
}

func LoadProviderDocFromFile(fileName string) (Provider, error) {
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return loadProviderDocFromBytes(bytes)
}

func GetProviderDocBytes(prov string) ([]byte, error) {
	fn, err := getProviderDoc(prov)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fn)
}

func getServiceDoc(url string) (io.ReadCloser, error) {
	return os.Open(path.Join(OpenapiFileRoot, url))
}

func getServiceDocBytes(url string) ([]byte, error) {
	f, err := getServiceDoc(url)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func GetResourcesRegisterDocBytes(url string) ([]byte, error) {
	return getServiceDocBytes(url)
}

func GetServiceDocBytes(url string) ([]byte, error) {
	return getServiceDocBytes(url)
}

func LoadProviderByName(prov, version string) (Provider, error) {
	b, err := GetProviderDocBytes(path.Join(prov, version))
	if err != nil {
		return nil, err
	}
	return LoadProviderDocFromBytes(b)
}

func findLatestDoc(serviceDir string) (string, error) {
	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		return "", err
	}
	var fileNames []string
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".sig") {
			fileNames = append(fileNames, entry.Name())
		}
	}
	fileCount := len(fileNames)
	if fileCount == 0 {
		return "", fmt.Errorf("no openapi files present in directory = '%s'", serviceDir)
	}
	sort.Strings(fileNames)
	return path.Join(serviceDir, fileNames[fileCount-1]), nil
}

func getLatestFile(entries []fs.DirEntry) (string, error) {
	var fileNames []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".json")) {
			fileNames = append(fileNames, entry.Name())
		}
	}
	fileCount := len(fileNames)
	if fileCount == 0 {
		return "", fmt.Errorf("getLatestFile() no openapi files present in directory")
	}
	sort.Strings(fileNames)
	return fileNames[fileCount-1], nil
}

// func findLatestEmbeddedDoc(fs embed.FS) (string, error) {
// 	entries, err := fs.ReadDir(".")
// 	if err != nil {
// 		return "", err
// 	}
// 	var fileNames []string
// 	for _, entry := range entries {
// 		if !entry.IsDir() {
// 			fileNames = append(fileNames, entry.Name())
// 		}
// 	}
// 	fileCount := len(fileNames)
// 	if fileCount == 0 {
// 		return "", fmt.Errorf("no openapi files present in directory = '%s'", serviceDir)
// 	}
// 	sort.Strings(fileNames)
// 	return path.Join(serviceDir, fileNames[fileCount-1]), nil
// }

func getProviderDoc(provider string) (string, error) {
	switch provider {
	case "google":
		return findLatestDoc(path.Join(OpenapiFileRoot, "googleapis.com"))
	}
	return findLatestDoc(path.Join(OpenapiFileRoot, provider))
}

func loadServiceDocFromBytes(ps ProviderService, bytes []byte) (Service, error) {
	loader := NewLoader()
	rv, err := loader.LoadFromBytes(bytes)
	if err != nil {
		return nil, err
	}
	prov, ok := ps.GetProvider()
	if !ok {
		return nil, fmt.Errorf("provider service '%s' does not have a provider", ps.GetID())
	}
	rv.setProvider(prov)
	rv.setProviderService(ps)
	err = loader.extractAndMergeQueryTransposeServiceLevel(rv)
	if err != nil {
		return nil, err
	}
	return rv, nil
}

func LoadServiceSubsetDocFromBytes(rr ResourceRegister, resourceKey string, bytes []byte) (Service, error) {
	loader := NewLoader()
	return loader.LoadFromBytesAndResources(rr, resourceKey, bytes)
}

func loadProviderDocFromBytes(bytes []byte) (Provider, error) {
	var prov standardProvider
	err := yaml.Unmarshal(bytes, &prov)
	if err != nil {
		return nil, err
	}
	for _, v := range prov.ProviderServices {
		v.setProvider(&prov)
	}
	return &prov, nil
}

func resourceregisterLoadBackwardsCompatibility(rr ResourceRegister) {
	sr := rr.GetServiceDocPath()
	for m, n := range rr.GetResources() {
		n.setProvider(rr.getProvider())
		n.setProviderService(rr.getProviderService())
		if n.GetServiceDocPath() != nil {
			sr = n.GetServiceDocPath()
		}
		for k, v := range n.GetMethods() {
			os := &v
			os.setProvider(rr.getProvider())
			os.setProviderService(rr.getProviderService())
			os.setResource(n)
			operationBackwardsCompatibility(os, sr)
			rr.setOpStore(m, k, os)
		}
	}
}

func operationBackwardsCompatibility(component OperationStore, sr *ServiceRef) {
	// backwards compatibility
	if component.GetPathRef() != nil {
		stub := "#/paths/"
		if sr != nil {
			stub = sr.Ref + "#/paths/"
		}
		component.setOperationRef(&OperationRef{
			Ref: stub + strings.ReplaceAll(component.GetPathRef().Ref, "/", "~1") + "/" + component.GetOperationRef().Ref,
		})
	}
	//
}

func (loader *standardLoader) resolveOperationRef(doc Service, rsc Resource, component OperationStore, pir *PathItemRef, sr *ServiceRef) (err error) {

	if component == nil {
		return errors.New("invalid operation: value MUST be an object")
	}

	if component.GetOperationRef() != nil && component.GetOperationRef().Value != nil {
		if loader.visitedOperation == nil {
			loader.visitedOperation = make(map[*openapi3.Operation]struct{})
		}
		if _, ok := loader.visitedOperation[component.GetOperationRef().Value]; ok {
			return nil
		}
		loader.visitedOperation[component.GetOperationRef().Value] = struct{}{}
	} else if component.GetStackQLConfig() != nil && component.GetStackQLConfig().GetViews() != nil {
		component.setService(doc)
		component.setProviderService(doc.getProviderService())
		component.setProvider(doc.getProvider())
		component.setResource(rsc)
		return nil
	}
	component.setService(doc)
	component.setProviderService(doc.getProviderService())
	component.setProvider(doc.getProvider())
	component.setResource(rsc)

	operationBackwardsCompatibility(component, sr)
	pk := component.GetOperationRef().ExtractPathItem()
	pi, ok := doc.getPath(pk)
	if !ok {
		return fmt.Errorf("could not extract path for '%s'", pk)
	}
	mk := component.GetOperationRef().extractMethodItem()

	ops := pi.Operations()
	if ops == nil {
		return fmt.Errorf("cannot find any operation for path = '%s'; nil operations", pk)
	}
	op, ok := ops[strings.ToUpper(mk)]
	if !ok {
		return fmt.Errorf("cannot find operation = '%s' for path = '%s'; missing operation", mk, pk)
	}

	component.setOperationRef(&OperationRef{Value: op, Ref: component.GetOperationRef().Ref})
	component.setPathItem(pi)
	err = loader.extractAndMergeQueryTransposeOpLevel(component)
	if err != nil {
		return err
	}
	return loader.extractAndMergeGraphQL(component)
}

func (loader *standardLoader) resolveContentDefault(content openapi3.Content, svc Service) (Schema, string, bool) {
	if content == nil {
		return nil, "", false
	}
	preferredMediaTypes := []string{"application/json", "application/xml", "application/octet-stream"}
	for _, mt := range preferredMediaTypes {
		rv, ok := content[mt]
		if ok && rv != nil && rv.Schema != nil && rv.Schema.Value != nil {
			return NewSchema(rv.Schema.Value, svc, rv.Schema.Ref, rv.Schema.Ref), mt, true
		}
	}
	return nil, "", false
}

func (loader *standardLoader) findBestResponseDefault(responses openapi3.Responses) (*openapi3.Response, bool) {
	var numericKeys []string
	for k := range responses {
		code, err := strconv.Atoi(k)
		if err == nil {
			if code < 300 {
				numericKeys = append(numericKeys, k)
			}
		}
	}
	if len(numericKeys) > 0 {
		sort.Strings(numericKeys)
		rv, ok := responses[numericKeys[0]]
		if ok && rv != nil && rv.Value != nil {
			return rv.Value, true
		}
	}
	rv, ok := responses["default"]
	if ok && rv != nil && rv.Value != nil {
		return rv.Value, true
	}
	return nil, false
}

func (loader *standardLoader) resolveExpectedRequest(doc Service, op *openapi3.Operation, component ExpectedRequest) (err error) {
	switch component.(type) {
	case nil:
		return nil
	}
	if component != nil && component.GetSchema() != nil {
		if loader.visitedExpectedRequest == nil {
			loader.visitedExpectedRequest = make(map[Schema]struct{})
		}
		if _, ok := loader.visitedExpectedRequest[component.GetSchema()]; ok {
			return nil
		}
		loader.visitedExpectedRequest[component.GetSchema()] = struct{}{}
	}

	if component == nil {
		return nil
	}
	bmt := component.GetBodyMediaType()
	if bmt != "" {
		if op.RequestBody == nil || op.RequestBody.Value == nil {
			return nil
		}
		sRef := op.RequestBody.Value.Content[bmt].Schema
		s := NewSchema(sRef.Value, doc, sRef.Ref, sRef.Ref)
		component.setSchema(s)
		return nil
	} else {
		sc, mt, ok := loader.resolveContentDefault(op.RequestBody.Value.Content, doc)
		if ok {
			component.setBodyMediaType(mt)
			component.setSchema(sc)
		}
	}

	return nil
}

func (loader *standardLoader) resolveSQLVerb(rsc Resource, component *OperationStoreRef, sqlVerb string) (err error) {
	if component != nil && component.hasValue() {
		if loader.visitedOperationStore == nil {
			loader.visitedOperationStore = make(map[OperationStore]struct{})
		}
		if _, ok := loader.visitedOperationStore[component.Value]; ok {
			return nil
		}
		loader.visitedOperationStore[component.Value] = struct{}{}
	}

	resolved, err := resolveSQLVerbFromResource(rsc, component, sqlVerb)
	if err != nil {
		return err
	}
	resolved.setSQLVerb(sqlVerb)
	component.Value = resolved
	if component.Value == nil {
		return fmt.Errorf("operation store ref not resolved")
	}
	return nil
}

func resolveSQLVerbFromResource(rsc Resource, component *OperationStoreRef, sqlVerb string) (*standardOperationStore, error) {

	if component == nil {
		return nil, fmt.Errorf("operation store ref not supplied")
	}
	osv, _, err := jsonpointer.GetForToken(rsc, component.Ref)
	if err != nil {
		return nil, err
	}
	resolved, ok := osv.(*standardOperationStore)
	if !ok {
		return nil, fmt.Errorf("operation store ref type '%T' not supported", osv)
	}
	if resolved.Inverse != nil && resolved.Inverse.OpRef.Ref != "" {
		val, _, err := jsonpointer.GetForToken(rsc, resolved.Inverse.OpRef.Ref)
		if err != nil {
			return nil, err
		}
		inverseOp, valOk := val.(*standardOperationStore)
		if !valOk {
			return nil, fmt.Errorf("operation store ref type '%T' not supported", val)
		}
		resolved.Inverse.OpRef.Value = inverseOp
	}
	rv := resolved
	rv.setSQLVerb(sqlVerb)
	return rv, nil
}

func latePassResolveInverse(svc Service, component *OperationStoreRef) (*standardOperationStore, error) {
	if component == nil || component.Value == nil {
		return nil, fmt.Errorf("late pass: operation store ref not supplied")
	}
	input := component.Value
	if input.Inverse != nil && input.Inverse.OpRef.Ref != "" {
		val, _, err := jsonpointer.GetForToken(svc, input.Inverse.OpRef.Ref)
		if err != nil {
			return nil, err
		}
		inverseOp, valOk := val.(*standardOperationStore)
		if !valOk {
			return nil, fmt.Errorf("operation store ref type '%T' not supported", val)
		}
		input.Inverse.OpRef.Value = inverseOp
	}
	return input, nil
}

func (loader *standardLoader) resolveExpectedResponse(doc Service, op *openapi3.Operation, component ExpectedResponse) (err error) {
	if component != nil && component.GetSchema() != nil {
		if loader.visitedExpectedResponse == nil {
			loader.visitedExpectedResponse = make(map[Schema]struct{})
		}
		if _, ok := loader.visitedExpectedResponse[component.GetSchema()]; ok {
			return nil
		}
		loader.visitedExpectedResponse[component.GetSchema()] = struct{}{}
	}

	if component == nil {
		return nil
	}
	bmt := component.GetBodyMediaType()
	ek := component.GetOpenAPIDocKey()
	if bmt != "" && ek != "" {
		ekObj, ok := op.Responses[ek]
		if !ok || ekObj.Value == nil || ekObj.Value.Content == nil || ekObj.Value.Content[bmt] == nil || ekObj.Value.Content[bmt].Schema == nil || ekObj.Value.Content[bmt].Schema.Value == nil {
			return nil
		}
		sRef := op.Responses[ek].Value.Content[bmt].Schema
		textualRepresentation := sRef.Ref
		if textualRepresentation == "" && sRef.Value.Items != nil && sRef.Value.Items.Ref != "" {
			textualRepresentation = fmt.Sprintf("[]%s", getPathSuffix(sRef.Value.Items.Ref))
		}
		s := NewSchema(sRef.Value, doc, textualRepresentation, sRef.Ref)
		component.setSchema(s)
		return nil
	} else {
		rs, ok := loader.findBestResponseDefault(op.Responses)
		if ok {
			sc, mt, ok := loader.resolveContentDefault(rs.Content, doc)
			if ok {
				component.setBodyMediaType(mt)
				component.setSchema(sc)
			}
		}
	}
	return nil
}
