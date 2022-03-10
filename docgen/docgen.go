package docgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/getkin/kin-openapi/openapi3"
)

type IResourcesExtractorStrategy interface {
	GetResources(*openapi3.T) (*openapistackql.ResourceRegister, error)
}

type BodyAttributes struct {
	BodySchema *openapi3.SchemaRef
	MediaType  string
}

type ResponseBodyAttributes struct {
	BodyAttributes
	ResponseCode string
}

type OperationDimensions struct {
	RequestBody  BodyAttributes
	ResponseBody ResponseBodyAttributes
}

type EndpointDimensions struct {
	OperationDimensions
	HttpVerb  string
	PathUrl   string
	Operation *openapi3.Operation
	PathItem  *openapi3.PathItem
}

func getSchemaName(sr *openapi3.SchemaRef) string {
	spl := strings.Split(sr.Ref, "/")
	if l := len(spl); l > 0 {
		return spl[l-1]
	}
	return ""
}

type BestProjectionFirstStrategy struct {
	rscMap map[string]*openapistackql.Resource
}

func NewNaiveResourcesExtractor() IResourcesExtractorStrategy {
	return &BestProjectionFirstStrategy{
		rscMap: make(map[string]*openapistackql.Resource),
	}
}

func (nr *BestProjectionFirstStrategy) getBestMediaType(c openapi3.Content) (string, error) {
	if c == nil {
		return "", fmt.Errorf("no content")
	}
	for _, k := range []string{"application/json", "application/xml", "application/yaml", "application/octet-stream"} {
		_, ok := c[k]
		if ok {
			return k, nil
		}
	}
	for k := range c {
		return k, nil
	}
	return "", fmt.Errorf("no content")
}

func (nr *BestProjectionFirstStrategy) getBestResponseCode(r openapi3.Responses) (string, error) {
	if r == nil || len(r) == 0 {
		return "", nil
	}
	var numericResponses []string
	for k := range r {
		if _, err := strconv.Atoi(k); err == nil {
			numericResponses = append(numericResponses, k)
		}
	}
	_, defaultOk := r["default"]
	if len(numericResponses) > 0 {
		sort.Strings(numericResponses)
		if numericResponses[0] < "300" {
			return numericResponses[0], nil
		}
		if defaultOk {
			return "default", nil
		}
		return numericResponses[0], nil
	}
	if defaultOk {
		return "default", nil
	}
	return "", fmt.Errorf("could not find an appopriate response")
}

func (nr *BestProjectionFirstStrategy) getBestRequest(op *openapi3.Operation) (BodyAttributes, error) {
	if op.RequestBody == nil || op.RequestBody.Value == nil || op.RequestBody.Value.Content == nil {
		return BodyAttributes{}, nil
	}
	k, err := nr.getBestMediaType(op.RequestBody.Value.Content)
	if err != nil {
		return BodyAttributes{}, err
	}
	ct := op.RequestBody.Value.Content[k]
	if ct == nil || ct.Schema == nil || op.RequestBody.Value.Content[k].Schema.Value == nil {
		return BodyAttributes{}, fmt.Errorf("inviable request body")
	}
	return BodyAttributes{
		BodySchema: op.RequestBody.Value.Content[k].Schema,
		MediaType:  k,
	}, nil
}

func (nr *BestProjectionFirstStrategy) getBestResponse(op *openapi3.Operation) (ResponseBodyAttributes, error) {

	s, err := nr.getBestResponseCode(op.Responses)

	if err != nil {
		return ResponseBodyAttributes{}, err
	}

	if s == "" {
		return ResponseBodyAttributes{}, nil
	}

	mt, ok := op.Responses[s]

	if !ok || mt == nil || mt.Value == nil || mt.Value.Content == nil || len(mt.Value.Content) == 0 {
		return ResponseBodyAttributes{}, fmt.Errorf("could not find response media types for key = '%s'", s)
	}

	bmt, err := nr.getBestMediaType(mt.Value.Content)

	if err != nil {
		return ResponseBodyAttributes{}, err
	}

	ss, ok := mt.Value.Content[bmt]

	if !ok || ss == nil || ss.Schema == nil || ss.Schema.Value == nil {
		return ResponseBodyAttributes{}, fmt.Errorf("no viable response")
	}

	return ResponseBodyAttributes{
		BodyAttributes: BodyAttributes{BodySchema: ss.Schema, MediaType: bmt},
		ResponseCode:   s,
	}, nil
}

func (nr *BestProjectionFirstStrategy) extractOperationDimensions(op *openapi3.Operation) (OperationDimensions, error) {
	requestBodySchema, err := nr.getBestRequest(op)
	if err != nil {
		return OperationDimensions{}, err
	}
	responseBodySchema, err := nr.getBestResponse(op)
	if err != nil {
		return OperationDimensions{}, err
	}
	return OperationDimensions{
		RequestBody:  requestBodySchema,
		ResponseBody: responseBodySchema,
	}, nil
}

func (nr *BestProjectionFirstStrategy) getEndpointDimensions(p *openapi3.PathItem, op *openapi3.Operation, pathString, verb string) (EndpointDimensions, bool, error) {
	dimz, err := nr.extractOperationDimensions(op)
	if err != nil {
		return EndpointDimensions{}, false, err
	}
	return EndpointDimensions{
		OperationDimensions: dimz,
		HttpVerb:            verb,
		PathUrl:             pathString,
		Operation:           op,
		PathItem:            p,
	}, true, nil
}

func (nr *BestProjectionFirstStrategy) getEndpointDimensionsSlice(t *openapi3.T) ([]EndpointDimensions, error) {
	var paths []EndpointDimensions
	for k, p := range t.Paths {
		for opk, op := range map[string]*openapi3.Operation{
			"CONNECT": p.Connect,
			"DELETE":  p.Delete,
			"GET":     p.Get,
			"HEAD":    p.Head,
			"OPTIONS": p.Options,
			"PATCH":   p.Patch,
			"POST":    p.Post,
			"PUT":     p.Put,
			"TRACE":   p.Trace,
		} {
			epd, ok, err := nr.getEndpointDimensions(p, op, k, opk)
			if err != nil {
				return nil, err
			}
			if ok {
				paths = append(paths, epd)
			}
		}
	}
	return paths, nil
}

func (nr *BestProjectionFirstStrategy) schemasToEmptyResourcesPass(t *openapi3.T, epDimSlice []EndpointDimensions) error {
	for k, s := range t.Components.Schemas {
		nr.rscMap[k] = &openapistackql.Resource{
			ID:          k,
			Name:        k,
			Title:       s.Value.Title,
			Description: s.Value.Description,
			Methods:     make(openapistackql.Methods),
		}
	}
	return nil
}

func (nr *BestProjectionFirstStrategy) getOpStore(s EndpointDimensions) openapistackql.OperationStore {
	opStore := openapistackql.OperationStore{
		PathItemRef: &openapistackql.PathItemRef{Ref: s.PathUrl},
		Request: &openapistackql.ExpectedRequest{
			BodyMediaType: s.ResponseBody.MediaType,
			Schema:        openapistackql.NewSchema(s.ResponseBody.BodySchema.Value, getSchemaName(s.ResponseBody.BodySchema)),
		},
	}
	if s.Operation.Servers != nil {
		opStore.Servers = s.Operation.Servers
	}
	return opStore
}

func (nr *BestProjectionFirstStrategy) naiveProjectionPass(t *openapi3.T, epDimSlice []EndpointDimensions) error {
	for _, s := range epDimSlice {
		if s.ResponseBody.BodySchema != nil {
			getOpStore := nr.getOpStore(s)
			sname := getSchemaName(s.ResponseBody.BodySchema)
			rsc, ok := nr.rscMap[sname]
			if ok {
				rsc.Methods["get"] = getOpStore
			}
			for pr, sr := range s.ResponseBody.BodySchema.Value.Properties {
				psname := getSchemaName(sr)
				selOpStore := nr.getOpStore(s)
				selOpStore.Response.ObjectKey = pr
				rsc, ok := nr.rscMap[psname]
				if ok {
					rsc.Methods["select"] = selOpStore
				}
			}
		}
	}
	return nil
}

func (nr *BestProjectionFirstStrategy) GetResources(t *openapi3.T) (*openapistackql.ResourceRegister, error) {
	epDimSlice, err := nr.getEndpointDimensionsSlice(t)
	if err != nil {
		return nil, err
	}
	if epDimSlice == nil {
		return nil, fmt.Errorf("no endpoint dimensions found")
	}
	return &openapistackql.ResourceRegister{
		Resources: nr.rscMap,
	}, nil
}

func GetDocs() {

}

/*

def _find_best_response(self, response_keys :dict_keys) -> str:
    rs = sorted([str(k) for k in response_keys])
    candidate = rs[0]
    if candidate.isnumeric() and int(candidate) < 300:
      return candidate
    elif 'default' in rs:
      return 'default'
    else:
      return candidate


  def get_resources(openapi_doc :dict) -> dict:
    for pk, pv in openapi_doc.get('paths', {}).items():
      for vk in pv.keys():
        best_response_key
      paths.append(str(p))
      for

    return {}

*/
