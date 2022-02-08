package openapistackql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	log "github.com/sirupsen/logrus"
)

func ProviderTypeConditionIsValid(providerType string, lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(providerType, lhs, rhs)

}

func providerTypeConditionIsValid(providerType string, lhs string, rhs interface{}) bool {
	switch providerType {
	case "string":
		return reflect.TypeOf(rhs).String() == "string"
	case "object":
		return false
	case "array":
		return false
	case "int", "int32", "int64":
		return reflect.TypeOf(rhs).String() == "int"
	default:
		return false
	}
	return false
}

func (s *Schema) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(s.Type, lhs, rhs)
}

type Schema struct {
	*openapi3.Schema
	key            string
	alwaysRequired bool
}

type Schemas map[string]*Schema

func NewSchema(sc *openapi3.Schema, key string) *Schema {
	var alwaysRequired bool
	if ar, ok := sc.Extensions[ExtensionKeyAlwaysRequired]; ok {
		if pr, ok := ar.(bool); ok && pr {
			alwaysRequired = true
		}
	}
	return &Schema{
		sc,
		key,
		alwaysRequired,
	}
}

func (s *Schema) GetProperties() (Schemas, error) {
	retVal := make(Schemas)
	for k, sr := range s.Properties {
		retVal[k] = NewSchema(sr.Value, k)
	}
	return retVal, nil
}

func getPathSuffix(path string) string {
	pathSplit := strings.Split(path, "/")
	return pathSplit[len(pathSplit)-1]
}

func (s *Schema) GetName() string {
	return getPathSuffix(s.key)
}

func (s *Schema) IsRequired(key string) bool {
	for _, req := range s.Required {
		if req == key {
			return true
		}
	}
	return false
}

func (s *Schema) GetItems() (*Schema, error) {
	if s.Items != nil && s.Items.Value != nil {
		itemsPathSplit := strings.Split(s.Items.Ref, "/")
		return NewSchema(s.Items.Value, itemsPathSplit[len(itemsPathSplit)-1]), nil
	}
	return nil, fmt.Errorf("no items present in schema with key = '%s'", s.key)
}

func (s *Schema) GetProperty(propertyKey string) (*Schema, error) {
	sc, ok := s.Properties[propertyKey]
	if !ok {
		return nil, fmt.Errorf("Schema.GetProperty() failure")
	}
	return NewSchema(sc.Value, propertyKey), nil
}

func (s *Schema) IsIntegral() bool {
	return s.Type == "int" || s.Type == "integer"
}

func (s *Schema) IsBoolean() bool {
	return s.Type == "bool" || s.Type == "boolean"
}

func (s *Schema) IsFloat() bool {
	return s.Type == "float" || s.Type == "float64"
}

func (sc *Schema) GetPropertySchema(key string) (*Schema, error) {
	absentErr := fmt.Errorf("property schema not present for key '%s'", key)
	sh, ok := sc.Properties[key]
	if !ok {
		return nil, absentErr
	}
	return NewSchema(
		sh.Value,
		key,
	), nil
}

func (sc *Schema) GetItemsSchema() (*Schema, error) {
	absentErr := fmt.Errorf("items schema not present")
	sh := sc.Items
	if sh.Value != nil {
		return NewSchema(
			sh.Value,
			"",
		), nil
	}
	return nil, absentErr
}

func (schema *Schema) GetSelectListItems(key string) (*Schema, string) {
	propS, ok := schema.Properties[key]
	if !ok {
		return nil, ""
	}
	itemS := propS.Value
	if itemS != nil {
		return NewSchema(
			itemS,
			"",
		), key
	}
	return nil, ""
}

func (schema *Schema) GetSelectSchema(itemsKey string) (*Schema, string, error) {
	sc, str, err := schema.getSelectItemsSchema(itemsKey)
	if err == nil {
		return sc, str, err
	}
	if schema != nil && schema.Properties != nil && len(schema.Properties) > 0 {
		return schema, "", nil
	}
	return nil, "", fmt.Errorf("unable to complete schema.GetSelectSchema() for schema = '%v' and itemsKey = '%s'", schema, itemsKey)
}

func (schema *Schema) getSelectItemsSchema(key string) (*Schema, string, error) {
	var itemS *openapi3.Schema
	log.Infoln(fmt.Sprintf("schema.getSelectItemsSchema() key = '%s'", key))
	if strings.HasPrefix(schema.key, "[]") {
		rv, err := schema.GetItems()
		return rv, key, err
	} else {
		propS, ok := schema.Properties[key]
		if !ok {
			return nil, "", fmt.Errorf("could not find items for key = '%s'", key)
		}
		itemS = propS.Value
	}
	if itemS != nil {
		s := NewSchema(
			itemS,
			key,
		)
		rv, err := s.GetItems()
		return rv, key, err
	}
	return nil, "", fmt.Errorf("could not find items for key = '%s'", key)
}

func (s *Schema) toFlatDescriptionMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	retVal["name"] = s.Title
	retVal["type"] = s.Type
	if extended {
		retVal["description"] = s.Description
	}
	return retVal
}

func (s *Schema) GetAllColumns() []string {
	log.Infoln(fmt.Sprintf("s = %v", *s))
	var retVal []string
	if s.Type == "object" || (s.Properties != nil && len(s.Properties) > 0) {
		for k, val := range s.Properties {
			valSchema := val.Value
			if valSchema != nil {
				retVal = append(retVal, k)
			}
		}
	} else if s.Type == "array" {
		if items := s.Items.Value; items != nil {
			iS := NewSchema(items, "")
			return iS.GetAllColumns()
		}
	}
	return retVal
}

func (s *Schema) IsArrayRef() bool {
	return s.Items != nil && s.Items.Value != nil
}

func (s *Schema) Tabulate(omitColumns bool) *Tabulation {
	if s.Type == "object" || (s.Properties != nil && len(s.Properties) > 0) {
		var cols []ColumnDescriptor
		if !omitColumns {
			for k, val := range s.Properties {
				valSchema := val.Value
				if valSchema != nil {
					col := ColumnDescriptor{Name: k, Schema: NewSchema(
						valSchema,
						k,
					)}
					cols = append(cols, col)
				}
			}
		}
		return &Tabulation{columns: cols, name: s.GetName()}
	} else if s.Type == "array" {
		if items := s.Items.Value; items != nil {

			return NewSchema(items, "").Tabulate(false)
		}
	} else if s.Type == "string" {
		cd := ColumnDescriptor{Name: "_", Schema: s}
		return &Tabulation{columns: []ColumnDescriptor{cd}, name: s.Title}
	}
	return nil
}

func (s *Schema) ToDescriptionMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	if s.Type == "array" {
		items := s.Items.Value
		if items != nil {
			return NewSchema(items, "").toFlatDescriptionMap(extended)
		}
	}
	if s.Type == "object" {
		for k, v := range s.Properties {
			p := v.Value
			if p != nil {
				pm := NewSchema(p, "").toFlatDescriptionMap(extended)
				pm["name"] = k
				retVal[k] = pm
			}
		}
		return retVal
	}
	retVal["name"] = s.Title
	retVal["type"] = s.Type
	if extended {
		retVal["description"] = s.Description
	}
	return retVal
}

func (s *Schema) FindByPath(path string, visited map[string]bool) *Schema {
	if visited == nil {
		visited = make(map[string]bool)
	}
	log.Infoln(fmt.Sprintf("FindByPath() called with path = '%s'", path))
	if s.key == path {
		return s
	}
	remainingPath := strings.TrimPrefix(path, s.key)
	if s.Type == "object" {
		for k, v := range s.Properties {
			if v.Ref != "" {
				isVis, ok := visited[v.Ref]
				if isVis && ok {
					continue
				}
				visited[v.Ref] = true
			}
			log.Infoln(fmt.Sprintf("FindByPath() attempting to match  path = '%s' with property '%s', visited = %v", path, k, visited))
			if k == path {
				rv := v.Value
				return NewSchema(rv, k)
			}
			ss := NewSchema(v.Value, k)
			if ss != nil {
				res := ss.FindByPath(path, visited)
				if res != nil {
					return res
				}
				resRem := ss.FindByPath(remainingPath, visited)
				if resRem != nil {
					return resRem
				}
			}
		}
	}
	if s.Type == "array" {
		ss, err := s.GetItems()
		if err != nil {
			return nil
		}
		return ss
	}
	return nil
}
