package queryrouter

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// An abstraction layer to provide a shim for functionality disallowed
// by the gorillamux router; eg. IPv4 addresses as server variabless
func NewRouter(doc *openapi3.T) (routers.Router, error) {
	return gorillamux.NewRouter(doc)
}
