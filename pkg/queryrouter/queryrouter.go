package queryrouter

/*
 * Abstraction layer for variations / extensions on
 * gorillamux routers as leveraged by "github.com/getkin/kin-openapi/openapi3":
 *   - Shim for routing with server variables.
 *   - Open for extension as per open/closed principle.
 *
 * This is highly derivative of "github.com/getkin/kin-openapi/openapi3",
 * and in particular https://github.com/getkin/kin-openapi/blob/c95dd68aef43fa9ac8c1f52f169b387d7681626a/routers/gorillamux/router.go
 *
 * "github.com/getkin/kin-openapi/openapi3" is
 * fully acknowledged (and greatly appreciated) in our licensing pages and README.
 */

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/gorilla/mux"

	"github.com/stackql/go-openapistackql/pkg/urltranslate"
)

var _ routers.Router = &Router{}

type Router struct {
	muxes  []*mux.Route
	routes []*routers.Route
}

func NewRouter(doc *openapi3.T) (routers.Router, error) {
	type srv struct {
		schemes []string
		host    urltranslate.QueryVar
		base    string
		server  *openapi3.Server
	}
	servers := make([]srv, 0, len(doc.Servers))
	for _, server := range doc.Servers {
		serverURLParameterised, err := urltranslate.ExtractParameterisedURL(server.URL)
		if err != nil {
			return nil, err
		}
		serverURL := serverURLParameterised.String()
		var schemes []string
		var u *url.URL
		if strings.Contains(serverURL, "://") {
			scheme0 := strings.Split(serverURL, "://")[0]
			schemes = permutePart(scheme0, server)
			u, err = url.Parse(bEncode(strings.Replace(serverURL, scheme0+"://", schemes[0]+"://", 1)))
		} else {
			u, err = url.Parse(bEncode(serverURL))
		}
		if err != nil {
			return nil, err
		}
		path := bDecode(u.EscapedPath())
		if len(path) > 0 && path[len(path)-1] == '/' {
			path = path[:len(path)-1]
		}
		hostVarName := strings.TrimSuffix(strings.TrimPrefix(bDecode(u.Host), "{"), "}")
		hostVar, ok := serverURLParameterised.GetVarByName(hostVarName)
		if !ok {
			return nil, fmt.Errorf("var = '%s' unavailable in URL = '%s'", hostVarName, serverURLParameterised.Raw())
		}
		servers = append(servers, srv{
			host:    hostVar, //u.Hostname()?
			base:    path,
			schemes: schemes, // scheme: []string{scheme0}, TODO: https://github.com/gorilla/mux/issues/624
			server:  server,
		})
	}
	if len(servers) == 0 {
		servers = append(servers, srv{})
	}
	muxRouter := mux.NewRouter().UseEncodedPath()
	r := &Router{}
	for _, path := range orderedPaths(doc.Paths) {
		pathItem := doc.Paths[path]

		operations := pathItem.Operations()
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		sort.Strings(methods)

		for _, s := range servers {
			muxRoute := muxRouter.Path(s.base + path).Methods(methods...)
			if strings.HasPrefix(path, "/?") {
				var pairs []string
				kvs := strings.Split(path[2:], "&")
				for _, v := range kvs {
					pairs = append(pairs, strings.Split(v, "=")...)
				}
				muxRoute = muxRouter.Queries(pairs...).Methods(methods...)
			}
			if schemes := s.schemes; len(schemes) != 0 {
				muxRoute.Schemes(schemes...)
			}
			if host := s.host; host != nil && host.GetName() != "" {
				muxRoute.Host(host.FullString())
			}
			if err := muxRoute.GetError(); err != nil {
				return nil, err
			}
			r.muxes = append(r.muxes, muxRoute)
			r.routes = append(r.routes, &routers.Route{
				Spec:      doc,
				Server:    s.server,
				Path:      path,
				PathItem:  pathItem,
				Method:    "",
				Operation: nil,
			})
		}
	}
	return r, nil
}

func (r *Router) FindRoute(req *http.Request) (*routers.Route, map[string]string, error) {
	for i, muxRoute := range r.muxes {
		var match mux.RouteMatch
		if muxRoute.Match(req, &match) {
			if err := match.MatchErr; err != nil {
				// What then?
			}
			route := *r.routes[i]
			route.Method = req.Method
			route.Operation = route.Spec.Paths[route.Path].GetOperation(route.Method)
			return &route, match.Vars, nil
		}
		switch match.MatchErr {
		case nil:
		case mux.ErrMethodMismatch:
			return nil, nil, routers.ErrMethodNotAllowed
		default: // What then?
		}
	}
	return nil, nil, routers.ErrPathNotFound
}

func orderedPaths(paths map[string]*openapi3.PathItem) []string {
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#pathsObject
	// When matching URLs, concrete (non-templated) paths would be matched
	// before their templated counterparts.
	// NOTE: sorting by number of variables ASC then by descending lexicographical
	// order seems to be a good heuristic.
	vars := make(map[int][]string)
	max := 0
	for path := range paths {
		count := strings.Count(path, "}")
		vars[count] = append(vars[count], path)
		if count > max {
			max = count
		}
	}
	ordered := make([]string, 0, len(paths))
	for c := 0; c <= max; c++ {
		if ps, ok := vars[c]; ok {
			sort.Sort(sort.Reverse(sort.StringSlice(ps)))
			ordered = append(ordered, ps...)
		}
	}
	return ordered
}

// Magic strings that temporarily replace "{}" so net/url.Parse() works
var blURL, brURL = strings.Repeat("-", 50), strings.Repeat("_", 50)

func bEncode(s string) string {
	s = strings.Replace(s, "{", blURL, -1)
	s = strings.Replace(s, "}", brURL, -1)
	return s
}
func bDecode(s string) string {
	s = strings.Replace(s, blURL, "{", -1)
	s = strings.Replace(s, brURL, "}", -1)
	return s
}

func permutePart(part0 string, srv *openapi3.Server) []string {
	type mapAndSlice struct {
		m map[string]struct{}
		s []string
	}
	var2val := make(map[string]mapAndSlice)
	max := 0
	for name0, v := range srv.Variables {
		name := "{" + name0 + "}"
		if !strings.Contains(part0, name) {
			continue
		}
		m := map[string]struct{}{v.Default: {}}
		for _, value := range v.Enum {
			m[value] = struct{}{}
		}
		if l := len(m); l > max {
			max = l
		}
		s := make([]string, 0, len(m))
		for value := range m {
			s = append(s, value)
		}
		var2val[name] = mapAndSlice{m: m, s: s}
	}
	if len(var2val) == 0 {
		return []string{part0}
	}

	partsMap := make(map[string]struct{}, max*len(var2val))
	for i := 0; i < max; i++ {
		part := part0
		for name, mas := range var2val {
			part = strings.Replace(part, name, mas.s[i%len(mas.s)], -1)
		}
		partsMap[part] = struct{}{}
	}
	parts := make([]string, 0, len(partsMap))
	for part := range partsMap {
		parts = append(parts, part)
	}
	sort.Strings(parts)
	return parts
}
