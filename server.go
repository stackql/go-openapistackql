package openapistackql

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func ObtainServerURLsFromServers(svs []*openapi3.Server, vars map[string]string) ([]string, error) {
	return obtainServerURLsFromServers(svs, vars)
}

func obtainServerURLsFromServers(svs []*openapi3.Server, vars map[string]string) ([]string, error) {
	var retVal []string
	if vars == nil {
		vars = make(map[string]string)
	}
	for _, sv := range svs {
		svUrl, err := generateServerURL(sv, vars)
		if err != nil {
			continue
		}
		retVal = append(retVal, svUrl)
	}
	if retVal == nil {
		return nil, fmt.Errorf("cannot find any viable servers")
	}
	return retVal, nil
}

func generateServerURL(sv *openapi3.Server, vars map[string]string) (string, error) {
	mergedVars := make(map[string]string)
	if vars == nil {
		vars = make(map[string]string)
	}
	for k, v := range sv.Variables {
		existing, alreadyExists := vars[k]
		if !alreadyExists {
			def := v.Default
			if def == "" {
				return "", fmt.Errorf("no default provided for server variable %s", k)
			}
			mergedVars[k] = def
		} else {
			mergedVars[k] = existing
		}
	}
	return replaceSimpleStringVars(sv.URL, mergedVars), nil
}

func replaceSimpleStringVars(template string, vars map[string]string) string {
	args := make([]string, len(vars)*2)
	i := 0
	for k, v := range vars {
		if strings.Contains(template, "{"+k+"}") {
			args[i] = "{" + k + "}"
			args[i+1] = v
			i += 2
		}
	}
	return strings.NewReplacer(args...).Replace(template)
}
