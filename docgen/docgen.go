package docgen

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/getkin/kin-openapi/openapi3"
)

type IResourcesExtractorStrategy interface {
	GetResources(*openapi3.T) (*openapistackql.ResourceRegister, error)
}

type EndpointDimensions struct {
	PathUrl   string
	Operation *openapi3.Operation
}

type BestProjectionFirstStrategy struct{}

func NewNaiveResourcesExtractor() *BestProjectionFirstStrategy {
	return &BestProjectionFirstStrategy{}
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

func (nr *BestProjectionFirstStrategy) getBestRequest(op *openapi3.Operation) *openapi3.Schema {
	if op.RequestBody == nil || op.RequestBody.Value == nil || op.RequestBody.Value.Content == nil {
		return nil
	}
	k, err := nr.getBestMediaType(op.RequestBody.Value.Content)
	if err != nil {
		return nil
	}
	ct := op.RequestBody.Value.Content[k]
	if ct == nil || ct.Schema == nil {
		return nil
	}
	return op.RequestBody.Value.Content[k].Schema.Value
}

func (nr *BestProjectionFirstStrategy) GetResources(t *openapi3.T) (*openapistackql.ResourceRegister, error) {
	var paths []EndpointDimensions
	for k, p := range t.Paths {
		if p.Delete != nil {

		}
		paths = append(paths, k)
	}
	return nil, nil
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
