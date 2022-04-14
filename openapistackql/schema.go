package openapistackql

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	log "github.com/sirupsen/logrus"
	"github.com/stackql/go-openapistackql/pkg/util"
)

const (
	defaultAnonymousColumnName string = "column_anon"
)

var (
	AnonymousColumnName string = defaultAnonymousColumnName
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
	if sc.Extensions != nil {
		if ar, ok := sc.Extensions[ExtensionKeyAlwaysRequired]; ok {
			if pr, ok := ar.(bool); ok && pr {
				alwaysRequired = true
			}
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
	if itemsKey == AnonymousColumnName {
		switch schema.Type {
		case "string", "integer":
			return schema, AnonymousColumnName, nil
		}
	}
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
	if strings.HasPrefix(schema.key, "[]") || schema.Type == "array" {
		rv, err := schema.GetItems()
		return rv, key, err
	} else if len(schema.Properties) > 0 {
		propS, ok := schema.Properties[key]
		if !ok {
			return nil, "", fmt.Errorf("could not find items for key = '%s'", key)
		}
		itemS = propS.Value
	} else if schema.hasPolymorphicProperties() {
		if len(schema.AllOf) > 0 {
			polySchema := getFatSchema(schema.AllOf)
			if polySchema == nil {
				return nil, "", fmt.Errorf("polymorphic select reposnse parse failed")
			}
			return polySchema, "", nil
		} else if len(schema.AnyOf) > 0 {
			polySchema := getFatSchema(schema.AnyOf)
			if polySchema == nil {
				return nil, "", fmt.Errorf("polymorphic select reposnse parse failed")
			}
			return polySchema, "", nil
		} else if len(schema.OneOf) > 0 {
			polySchema := getFatSchema(schema.OneOf)
			if polySchema == nil {
				return nil, "", fmt.Errorf("polymorphic select reposnse parse failed")
			}
			return polySchema, "", nil
		} else {
			return nil, "", fmt.Errorf("polymorphic select reposnse parse failed")
		}
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
	switch s.Type {
	case "string", "bool", "integer":
		return []string{AnonymousColumnName}
	}
	return retVal
}

func (s *Schema) IsArrayRef() bool {
	return s.Items != nil && s.Items.Value != nil
}

func (s *Schema) getPropertiesColumns() []ColumnDescriptor {
	var cols []ColumnDescriptor
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
	return cols
}

func (s *Schema) getAllOfColumns() []ColumnDescriptor {
	return s.getAllSchemaRefsColumns(s.AllOf)
}

func (s *Schema) getAnyOfColumns() []ColumnDescriptor {
	return s.getAllSchemaRefsColumns(s.AnyOf)
}

func (s *Schema) getOneOfColumns() []ColumnDescriptor {
	return s.getAllSchemaRefsColumns(s.OneOf)
}

func getSchemaName(sr *openapi3.SchemaRef) string {
	spl := strings.Split(sr.Ref, "/")
	if l := len(spl); l > 0 {
		return spl[l-1]
	}
	return ""
}

func getFatSchema(srs openapi3.SchemaRefs) *Schema {
	var rv *Schema
	for k, val := range srs {
		log.Debugf("processing composite key number = %d, id = '%s'\n", k, val.Ref)
		ss := NewSchema(val.Value, "")
		if rv == nil {
			rv = ss
			continue
		}
		for k, sRef := range ss.Properties {
			_, alreadyExists := rv.Properties[k]
			if alreadyExists {
				cn := fmt.Sprintf("%s_%s", getSchemaName(val), k)
				rv.Properties[cn] = sRef
				continue
			}
			rv.Properties[k] = sRef
		}
	}
	return rv
}

func (s *Schema) getAllSchemaRefsColumns(srs openapi3.SchemaRefs) []ColumnDescriptor {
	sc := getFatSchema(srs)
	st := sc.Tabulate(false)
	return st.GetColumns()
}

func (s *Schema) hasPolymorphicProperties() bool {
	if len(s.AllOf) > 0 || len(s.AnyOf) > 0 || len(s.OneOf) > 0 {
		return true
	}
	return false
}

func (s *Schema) hasPropertiesOrPolymorphicProperties() bool {
	if s.Properties != nil && len(s.Properties) > 0 {
		return true
	}
	return s.hasPolymorphicProperties()
}

func (s *Schema) Tabulate(omitColumns bool) *Tabulation {
	if s.Type == "object" || s.hasPropertiesOrPolymorphicProperties() {
		var cols []ColumnDescriptor
		if !omitColumns {
			if len(s.Properties) > 0 {
				cols = s.getPropertiesColumns()
			} else if len(s.AllOf) > 0 {
				cols = s.getAllOfColumns()
			} else if len(s.AnyOf) > 0 {
				cols = s.getAnyOfColumns()
			} else if len(s.OneOf) > 0 {
				cols = s.getOneOfColumns()
			}
		}
		return &Tabulation{columns: cols, name: s.GetName()}
	} else if s.Type == "array" {
		if items := s.Items.Value; items != nil {

			return NewSchema(items, "").Tabulate(omitColumns)
		}
	} else if s.Type == "string" {
		cd := ColumnDescriptor{Name: AnonymousColumnName, Schema: s}
		if omitColumns {
			return &Tabulation{columns: []ColumnDescriptor{}, name: s.Title}
		}
		return &Tabulation{columns: []ColumnDescriptor{cd}, name: s.Title}
	}
	return nil
}

func (s *Schema) ToDescriptionMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	if s.Type == "array" {
		items := s.Items.Value
		if items != nil {
			return NewSchema(items, "").ToDescriptionMap(extended)
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
	atomicMap := s.toFlatDescriptionMap(extended)
	atomicMap["name"] = AnonymousColumnName
	retVal[AnonymousColumnName] = atomicMap
	return retVal
}

func (s *Schema) getFattnedPolymorphicSchema() *Schema {
	if len(s.AllOf) > 0 {
		return getFatSchema(s.AllOf)
	}
	if len(s.OneOf) > 0 {
		return getFatSchema(s.OneOf)
	}
	if len(s.AnyOf) > 0 {
		return getFatSchema(s.AnyOf)
	}
	return nil
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
	if s.Type == "object" || s.hasPropertiesOrPolymorphicProperties() {
		if s.hasPolymorphicProperties() {
			fs := s.getFattnedPolymorphicSchema()
			return fs.FindByPath(path, visited)
		}
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

func (s *Schema) ProcessHttpResponse(response *http.Response) (interface{}, error) {
	target, err := marshalResponse(response)
	if err == nil && response.StatusCode >= 400 {
		err = fmt.Errorf(fmt.Sprintf("HTTP response error: %s", string(util.InterfaceToBytes(target, true))))
	}
	if err == io.EOF {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return map[string]interface{}{"result": "The Operation Completed Successfully"}, nil
		}
	}
	switch rv := target.(type) {
	case string, int:
		return map[string]interface{}{AnonymousColumnName: []interface{}{rv}}, nil
	}
	return target, err
}

func (s *Schema) DeprecatedProcessHttpResponse(response *http.Response) (map[string]interface{}, error) {
	target, err := s.ProcessHttpResponse(response)
	if err != nil {
		return nil, err
	}
	switch rv := target.(type) {
	case map[string]interface{}:
		return rv, nil
	case nil:
		return nil, nil
	case string:
		return map[string]interface{}{AnonymousColumnName: rv}, nil
	case []byte:
		return map[string]interface{}{AnonymousColumnName: string(rv)}, nil
	default:
		return nil, fmt.Errorf("DeprecatedProcessHttpResponse() cannot acccept response of type %T", rv)
	}
}
