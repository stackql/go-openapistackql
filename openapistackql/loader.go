package openapistackql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	yamlconv "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
)

const (
	ConfigFilesMode fs.FileMode = 0664
)

var (
	OpenapiFileRoot string
	IgnoreEmbedded  bool
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
	visitedPathItem         map[*openapi3.PathItem]struct{}
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

	for _, rsc := range rscMap {
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
	}
	svc.rsc = rscMap
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
	if !IgnoreEmbedded {
		switch prov {
		case "google":
			entries, err := googleProvider.ReadDir("embeddedproviders/googleapis.com")
			if err != nil {
				return nil, fmt.Errorf("wtf: %s", err.Error())
			}
			fn, err := getLatestFile(entries)
			if err != nil {
				return nil, fmt.Errorf("huh: %s", err.Error())
			}
			return googleProvider.ReadFile(path.Join("embeddedproviders/googleapis.com", fn))
		case "okta":
			entries, err := oktaProvider.ReadDir("embeddedproviders/okta")
			if err != nil {
				return nil, fmt.Errorf("wtf: %s", err.Error())
			}
			fn, err := getLatestFile(entries)
			if err != nil {
				return nil, fmt.Errorf("huh: %s", err.Error())
			}
			return oktaProvider.ReadFile(path.Join("embeddedproviders/okta", fn))
		}
	}
	fn, err := getProviderDoc(prov)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fn)
}

func GetServiceDocBytes(url string) ([]byte, error) {
	if !IgnoreEmbedded {
		pathElems := strings.Split(url, "/")
		prov := pathElems[0]
		// svc := pathElems[1]
		switch prov {
		case "google", "googleapis.com":
			// entries, err := googleProvider.ReadDir(path.Join("embeddedproviders/googleapis.com", svc))
			// if err != nil {
			// 	return nil, fmt.Errorf("wtf: %s", err.Error())
			// }
			// fn, err := getLatestFile(entries)
			// if err != nil {
			// 	return nil, fmt.Errorf("huh: %s", err.Error())
			// }
			return googleProvider.ReadFile(path.Join("embeddedproviders", strings.ReplaceAll(url, "/google/", "/googleapis.com/")))
		case "okta":
			return oktaProvider.ReadFile(path.Join("embeddedproviders", url))
		}
	}
	return os.ReadFile(path.Join(OpenapiFileRoot, url))
}

func LoadProviderByName(provider string) (*Provider, error) {
	b, err := GetProviderDocBytes(provider)
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
		if !entry.IsDir() {
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
		if !entry.IsDir() {
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
