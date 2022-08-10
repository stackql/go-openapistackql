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
)

func init() {
	OpenapiFileRoot = "."
}

type DiscoveryDoc interface {
	iDiscoveryDoc()
}

type Loader struct {
	*openapi3.Loader
	//
	visitedExpectedRequest  map[*Schema]struct{}
	visitedExpectedResponse map[*Schema]struct{}
	visitedOperation        map[*openapi3.Operation]struct{}
	visitedOperationStore   map[*OperationStore]struct{}
	visitedPathItem         map[*openapi3.PathItem]struct{}
}

func LoadResourcesShallow(ps *ProviderService, bt []byte) (*ResourceRegister, error) {
	return loadResourcesShallow(ps, bt)
}

func loadResourcesShallow(ps *ProviderService, bt []byte) (*ResourceRegister, error) {
	rv := NewResourceRegister()
	err := yaml.Unmarshal(bt, &rv)
	if err != nil {
		return nil, err
	}
	rv.Provider = ps.Provider
	rv.ProviderService = ps
	resourceregisterLoadBackwardsCompatibility(rv)
	return rv, nil
}

func (l *Loader) LoadFromBytes(bytes []byte) (*Service, error) {
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

func (l *Loader) LoadFromBytesAndResources(rr *ResourceRegister, resourceKey string, bytes []byte) (*Service, error) {
	doc, err := l.LoadFromData(bytes)
	if err != nil {
		return nil, err
	}
	svc := NewService(doc)
	docUrl := rr.ObtainServiceDocUrl(resourceKey)
	if docUrl != "" {
		err = l.mergeResourcesScoped(svc, docUrl, rr)
	} else {
		err = l.mergeResources(svc, rr.Resources, rr.ServiceDocPath)
	}
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (l *Loader) extractResources(svc *Service) error {
	rscs, ok := svc.Components.Extensions[ExtensionKeyResources]
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
	rscMap := make(map[string]*Resource)
	err = yaml.Unmarshal(bt, rscMap)
	if err != nil {
		return err
	}
	return l.mergeResources(svc, rscMap, nil)
}

func (l *Loader) extractAndMergeGraphQL(operation *OperationStore) error {
	if operation.OperationRef == nil || operation.OperationRef.Value == nil {
		return nil
	}
	gql, ok := operation.OperationRef.Value.Extensions[ExtensionKeyGraphQL]
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
	var rv GraphQL
	err = yaml.Unmarshal(bt, &rv)
	if err != nil {
		return err
	}
	operation.GraphQL = &rv
	return nil
}

func (l *Loader) mergeResources(svc *Service, rscMap map[string]*Resource, sdRef *ServiceRef) error {
	for _, rsc := range rscMap {
		var sr *ServiceRef
		if sdRef != nil {
			sr = sdRef
		}
		if rsc.ServiceDocPath != nil {
			sr = rsc.ServiceDocPath
		}
		err := l.mergeResource(svc, rsc, sr)
		if err != nil {
			return err
		}
	}
	svc.rsc = rscMap
	return nil
}

func (l *Loader) mergeResourcesScoped(svc *Service, svcUrl string, rr *ResourceRegister) error {
	scopedMap := make(map[string]*Resource)
	for k, rsc := range rr.Resources {
		if rr.ObtainServiceDocUrl(k) == svcUrl {
			err := l.mergeResource(svc, rsc, &ServiceRef{Ref: svcUrl})
			if err != nil {
				return err
			}
			scopedMap[k] = rsc
		}
	}
	if svc.rsc == nil {
		svc.rsc = scopedMap
		return nil
	}
	for k, v := range scopedMap {
		svc.rsc[k] = v
	}
	return nil
}

func (l *Loader) mergeResource(svc *Service, rsc *Resource, sr *ServiceRef) error {
	for k, vOp := range rsc.Methods {
		v := vOp
		v.MethodKey = k
		err := l.resolveOperationRef(svc, rsc, &v, v.PathRef, sr)
		if err != nil {
			return err
		}
		if v.Request == nil && v.OperationRef.Value.RequestBody != nil {
			v.Request = &ExpectedRequest{}
		}
		err = l.resolveExpectedRequest(svc, v.OperationRef.Value, v.Request)
		if err != nil {
			return err
		}
		err = l.resolveExpectedResponse(svc, v.OperationRef.Value, v.Response)
		if err != nil {
			return err
		}
		v.Servers = &svc.Servers
		rsc.Methods[k] = v
	}
	for sqlVerb, dir := range rsc.SQLVerbs {
		for i, v := range dir {
			cur := v
			err := l.resolveSQLVerb(rsc, &cur, sqlVerb)
			if err != nil {
				return err
			}
			rsc.SQLVerbs[sqlVerb][i] = cur
		}
	}
	return nil
}

func (svc *Service) ToJson() ([]byte, error) {
	return svc.MarshalJSON()
}

func (svc *Service) ToYaml() ([]byte, error) {
	j, err := svc.ToJson()
	if err != nil {
		return nil, err
	}
	return yamlconv.JSONToYAML(j)
}

func (pr *Provider) ToJson() ([]byte, error) {
	return pr.MarshalJSON()
}

func (pr *Provider) ToYaml() ([]byte, error) {
	j, err := pr.ToJson()
	if err != nil {
		return nil, err
	}
	return yamlconv.JSONToYAML(j)
}

func (svc *Service) ToYamlFile(filePath string) error {
	bytes, err := svc.ToYaml()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, ConfigFilesMode)
}

func (pr *Provider) ToYamlFile(filePath string) error {
	bytes, err := pr.ToYaml()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, ConfigFilesMode)
}

func NewLoader() *Loader {
	return &Loader{
		&openapi3.Loader{Context: context.Background()},
		make(map[*Schema]struct{}),
		make(map[*Schema]struct{}),
		make(map[*openapi3.Operation]struct{}),
		make(map[*OperationStore]struct{}),
		make(map[*openapi3.PathItem]struct{}),
	}
}

func LoadServiceDocFromBytes(ps *ProviderService, bytes []byte) (*Service, error) {
	return loadServiceDocFromBytes(ps, bytes)
}

func LoadProviderDocFromBytes(bytes []byte) (*Provider, error) {
	return loadProviderDocFromBytes(bytes)
}

func LoadServiceDocFromFile(ps *ProviderService, fileName string) (*Service, error) {
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return loadServiceDocFromBytes(ps, bytes)
}

func LoadProviderDocFromFile(fileName string) (*Provider, error) {
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

func LoadProviderByName(prov, version string) (*Provider, error) {
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

func loadServiceDocFromBytes(ps *ProviderService, bytes []byte) (*Service, error) {
	loader := NewLoader()
	rv, err := loader.LoadFromBytes(bytes)
	if err != nil {
		return nil, err
	}
	rv.Provider = ps.Provider
	rv.ProviderService = ps
	return rv, nil
}

func LoadServiceSubsetDocFromBytes(rr *ResourceRegister, resourceKey string, bytes []byte) (*Service, error) {
	loader := NewLoader()
	return loader.LoadFromBytesAndResources(rr, resourceKey, bytes)
}

func loadProviderDocFromBytes(bytes []byte) (*Provider, error) {
	var prov Provider
	err := yaml.Unmarshal(bytes, &prov)
	if err != nil {
		return nil, err
	}
	for _, v := range prov.ProviderServices {
		v.Provider = &prov
	}
	return &prov, nil
}

func resourceregisterLoadBackwardsCompatibility(rr *ResourceRegister) {
	sr := rr.ServiceDocPath
	for m, n := range rr.Resources {
		n.Provider = rr.Provider
		n.ProviderService = rr.ProviderService
		if n.ServiceDocPath != nil {
			sr = n.ServiceDocPath
		}
		for k, v := range n.Methods {
			os := v
			os.Provider = rr.Provider
			os.ProviderService = rr.ProviderService
			os.Resource = n
			operationBackwardsCompatibility(&os, sr)
			rr.Resources[m].Methods[k] = os
		}
	}
}

func operationBackwardsCompatibility(component *OperationStore, sr *ServiceRef) {
	// backwards compatibility
	if component.PathRef != nil {
		stub := "#/paths/"
		if sr != nil {
			stub = sr.Ref + "#/paths/"
		}
		component.OperationRef = &OperationRef{
			Ref: stub + strings.ReplaceAll(component.PathRef.Ref, "/", "~1") + "/" + component.OperationRef.Ref,
		}
	}
	//
}

func (loader *Loader) resolveOperationRef(doc *Service, rsc *Resource, component *OperationStore, pir *PathItemRef, sr *ServiceRef) (err error) {
	if component.OperationRef != nil && component.OperationRef.Value != nil {
		if loader.visitedOperation == nil {
			loader.visitedOperation = make(map[*openapi3.Operation]struct{})
		}
		if _, ok := loader.visitedOperation[component.OperationRef.Value]; ok {
			return nil
		}
		loader.visitedOperation[component.OperationRef.Value] = struct{}{}
	}

	if component == nil {
		return errors.New("invalid operation: value MUST be an object")
	}
	operationBackwardsCompatibility(component, sr)
	pk := component.OperationRef.ExtractPathItem()
	pi, ok := doc.Paths[pk]
	if !ok {
		return fmt.Errorf("could not extract path for '%s'", pk)
	}
	mk := component.OperationRef.extractMethodItem()

	ops := pi.Operations()
	if ops == nil {
		return fmt.Errorf("cannot find any operation for path = '%s'; nil operations", pk)
	}
	op, ok := ops[strings.ToUpper(mk)]
	if !ok {
		return fmt.Errorf("cannot find operation = '%s' for path = '%s'; missing operation", mk, pk)
	}

	component.OperationRef.Value = op
	component.PathItem = pi
	return loader.extractAndMergeGraphQL(component)
}

func (loader *Loader) resolveContentDefault(content openapi3.Content) (*Schema, string, bool) {
	if content == nil {
		return nil, "", false
	}
	preferredMediaTypes := []string{"application/json", "application/xml", "application/octet-stream"}
	for _, mt := range preferredMediaTypes {
		rv, ok := content[mt]
		if ok && rv != nil && rv.Schema != nil && rv.Schema.Value != nil {
			return NewSchema(rv.Schema.Value, rv.Schema.Ref), mt, true
		}
	}
	return nil, "", false
}

func (loader *Loader) findBestResponseDefault(responses openapi3.Responses) (*openapi3.Response, bool) {
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

func (loader *Loader) GetDocBytes(responses openapi3.Responses) (*Schema, bool) {
	if responses == nil {
		return nil, false
	}

	r, ok := loader.findBestResponseDefault(responses)
	if !ok || r == nil {
		return nil, false
	}
	sc, _, err := loader.resolveContentDefault(r.Content)
	return sc, err
}

func (loader *Loader) resolveExpectedRequest(doc *Service, op *openapi3.Operation, component *ExpectedRequest) (err error) {
	if component != nil && component.Schema != nil {
		if loader.visitedExpectedRequest == nil {
			loader.visitedExpectedRequest = make(map[*Schema]struct{})
		}
		if _, ok := loader.visitedExpectedRequest[component.Schema]; ok {
			return nil
		}
		loader.visitedExpectedRequest[component.Schema] = struct{}{}
	}

	if component == nil {
		return nil
	}
	bmt := component.BodyMediaType
	if bmt != "" {
		if op.RequestBody == nil || op.RequestBody.Value == nil {
			return nil
		}
		sRef := op.RequestBody.Value.Content[bmt].Schema
		s := NewSchema(sRef.Value, sRef.Ref)
		component.Schema = s
		return nil
	} else {
		sc, mt, ok := loader.resolveContentDefault(op.RequestBody.Value.Content)
		if ok {
			component.BodyMediaType = mt
			component.Schema = sc
		}
	}

	return nil
}

func (loader *Loader) resolveSQLVerb(rsc *Resource, component *OperationStoreRef, sqlVerb string) (err error) {
	if component != nil && component.Value != nil {
		if loader.visitedOperationStore == nil {
			loader.visitedOperationStore = make(map[*OperationStore]struct{})
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
	resolved.SQLVerb = sqlVerb
	component.Value = resolved
	if component.Value == nil {
		return fmt.Errorf("operation store ref not resolved")
	}
	return nil
}

func resolveSQLVerbFromResource(rsc *Resource, component *OperationStoreRef, sqlVerb string) (*OperationStore, error) {

	if component == nil {
		return nil, fmt.Errorf("operation store ref not supplied")
	}
	osv, _, err := jsonpointer.GetForToken(rsc, component.Ref)
	if err != nil {
		return nil, err
	}
	resolved, ok := osv.(*OperationStore)
	if !ok {
		return nil, fmt.Errorf("operation store ref type '%T' not supported", osv)
	}
	if resolved == nil {
		return nil, fmt.Errorf("operation store ref not resolved")
	}
	resolved.SQLVerb = sqlVerb
	return resolved, nil
}

func (loader *Loader) resolveExpectedResponse(doc *Service, op *openapi3.Operation, component *ExpectedResponse) (err error) {
	if component != nil && component.Schema != nil {
		if loader.visitedExpectedResponse == nil {
			loader.visitedExpectedResponse = make(map[*Schema]struct{})
		}
		if _, ok := loader.visitedExpectedResponse[component.Schema]; ok {
			return nil
		}
		loader.visitedExpectedResponse[component.Schema] = struct{}{}
	}

	if component == nil {
		return nil
	}
	bmt := component.BodyMediaType
	ek := component.OpenAPIDocKey
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
		s := NewSchema(sRef.Value, textualRepresentation)
		component.Schema = s
		return nil
	} else {
		rs, ok := loader.findBestResponseDefault(op.Responses)
		if ok {
			sc, mt, ok := loader.resolveContentDefault(rs.Content)
			if ok {
				component.BodyMediaType = mt
				component.Schema = sc
			}
		}
	}
	return nil
}
