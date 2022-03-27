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

func LoadResourcesShallow(bt []byte) (*ResourceRegister, error) {
	return loadResourcesShallow(bt)
}

func loadResourcesShallow(bt []byte) (*ResourceRegister, error) {
	rv := NewResourceRegister()
	err := yaml.Unmarshal(bt, &rv)
	if err != nil {
		return nil, err
	}
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
		err = l.mergeResources(svc, rr.Resources)
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
	return l.mergeResources(svc, rscMap)
}

func (l *Loader) mergeResources(svc *Service, rscMap map[string]*Resource) error {
	for _, rsc := range rscMap {
		err := l.mergeResource(svc, rsc)
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
			err := l.mergeResource(svc, rsc)
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

func (l *Loader) mergeResource(svc *Service, rsc *Resource) error {
	for k, v := range rsc.Methods {
		v.MethodKey = k
		err := l.resolvePathItemRef(svc, v.PathItemRef)
		if err != nil {
			return err
		}
		err = l.resolveOperationRef(svc, v.OperationRef, v.PathItemRef.Ref)
		if err != nil {
			return err
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
			err := l.resolveSQLVerb(rsc, &cur)
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

func LoadServiceDocFromBytes(bytes []byte) (*Service, error) {
	return loadServiceDocFromBytes(bytes)
}

func LoadProviderDocFromBytes(bytes []byte) (*Provider, error) {
	return loadProviderDocFromBytes(bytes)
}

func LoadServiceDocFromFile(fileName string) (*Service, error) {
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return loadServiceDocFromBytes(bytes)
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

func loadServiceDocFromBytes(bytes []byte) (*Service, error) {
	loader := NewLoader()
	return loader.LoadFromBytes(bytes)
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
	return &prov, nil
}

func (loader *Loader) resolveOperationRef(doc *Service, component *OperationRef, path string) (err error) {
	if component != nil && component.Value != nil {
		if loader.visitedOperation == nil {
			loader.visitedOperation = make(map[*openapi3.Operation]struct{})
		}
		if _, ok := loader.visitedOperation[component.Value]; ok {
			return nil
		}
		loader.visitedOperation[component.Value] = struct{}{}
	}

	if component == nil {
		return errors.New("invalid operation: value MUST be an object")
	}
	ref := component.Ref
	if ref != "" {
		p, ok := doc.Paths[path]
		if !ok {
			return fmt.Errorf("cannot find path = '%s'", path)
		}
		ops := p.Operations()
		if ops == nil {
			return fmt.Errorf("cannot find any operation for path = '%s'; nil operations", path)
		}
		op, ok := ops[strings.ToUpper(component.Ref)]
		if !ok {
			return fmt.Errorf("cannot find operation = '%s' for path = '%s'; missing operation", component.Ref, path)
		}
		component.Value = op
	}
	return nil
}

func (loader *Loader) resolvePathItemRef(doc *Service, component *PathItemRef) (err error) {
	if component != nil && component.Value != nil {
		if loader.visitedPathItem == nil {
			loader.visitedPathItem = make(map[*openapi3.PathItem]struct{})
		}
		if _, ok := loader.visitedPathItem[component.Value]; ok {
			return nil
		}
		loader.visitedPathItem[component.Value] = struct{}{}
	}

	if component == nil {
		return errors.New("invalid operation: value MUST be an object")
	}
	ref := component.Ref
	if ref != "" {
		p, ok := doc.Paths[ref]
		if !ok {
			return fmt.Errorf("cannot find path = '%s'", ref)
		}
		component.Value = p
	}
	return nil
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
	}
	return nil
}

func (loader *Loader) resolveSQLVerb(rsc *Resource, component *OperationStoreRef) (err error) {
	if component != nil && component.Value != nil {
		if loader.visitedOperationStore == nil {
			loader.visitedOperationStore = make(map[*OperationStore]struct{})
		}
		if _, ok := loader.visitedOperationStore[component.Value]; ok {
			return nil
		}
		loader.visitedOperationStore[component.Value] = struct{}{}
	}

	if component == nil {
		return fmt.Errorf("operation store ref not supplied")
	}
	osv, _, err := jsonpointer.GetForToken(rsc, component.Ref)
	if err != nil {
		return err
	}
	resolved, ok := osv.(*OperationStore)
	if !ok {
		return fmt.Errorf("operation store ref type '%T' not supported", osv)
	}
	component.Value = resolved
	if component.Value == nil {
		return fmt.Errorf("operation store ref not resolved")
	}
	return nil
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
	}
	return nil
}
