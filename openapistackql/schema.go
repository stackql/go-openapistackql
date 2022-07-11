package openapistackql

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/antchfx/xmlquery"
	"github.com/getkin/kin-openapi/openapi3"
	log "github.com/sirupsen/logrus"
	"github.com/stackql/go-openapistackql/pkg/media"
	"github.com/stackql/go-openapistackql/pkg/openapitopath"
	"github.com/stackql/go-openapistackql/pkg/response"
	"github.com/stackql/go-openapistackql/pkg/util"
	"github.com/stackql/go-openapistackql/pkg/xmlmap"
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
	return s.getProperties(), nil
}

func (s *Schema) getProperties() Schemas {
	retVal := make(Schemas)
	if s.hasPolymorphicProperties() {
		ss := s.getFattnedPolymorphicSchema()
		if ss != nil {
			for k, sr := range ss.Properties {
				retVal[k] = NewSchema(sr.Value, k)
			}
		}
	}
	for k, sr := range s.Properties {
		retVal[k] = NewSchema(sr.Value, k)
	}
	return retVal
}

func getPathSuffix(path string) string {
	pathSplit := strings.Split(path, "/")
	return pathSplit[len(pathSplit)-1]
}

func (s *Schema) GetName() string {
	return s.getName()
}

func (s *Schema) getName() string {
	return getPathSuffix(s.key)
}

func (s *Schema) getXMLALiasOrName() string {
	xa := s.getXmlAlias()
	if xa != "" {
		return xa
	}
	return s.getName()
}

func (s *Schema) IsRequired(key string) bool {
	for _, req := range s.Required {
		if req == key {
			return true
		}
	}
	return false
}

func (s *Schema) getXMLChild(path string) (*Schema, bool) {
	xmlAlias := s.getXmlAlias()
	if xmlAlias == path {
		return s, true
	}
	for _, v := range s.getProperties() {
		if v.getXmlAlias() == path {
			return v, true
		}
	}
	for _, v := range s.AllOf {
		if v.Value == nil {
			continue
		}
		si := v.Value
		if si.Type == "array" && si.Items != nil && si.Items.Value != nil {
			ss := NewSchema(si.Items.Value, "")
			_, ok := ss.getXMLChild(path)
			if ok {
				return NewSchema(si, ""), true
			}
			return nil, false
		}
	}
	return nil, false
}

func (s *Schema) getXMLDescendentInit(path []string) (*Schema, bool) {
	if len(path) == 0 {
		return s, true
	}
	if s.Type == "object" && len(path) > 0 {
		path = path[1:]
	}
	if len(path) == 0 {
		return s, true
	}
	p, ok := s.getProperty(path[0])
	if !ok {
		p, ok = s.getXMLChild(path[0])
		if !ok {
			return nil, false
		}
	}
	return p.getXMLDescendent(path[1:])
}

func (s *Schema) getDescendentInit(path []string) (*Schema, bool) {
	if len(path) == 0 {
		return s, true
	}
	if s.Type == "object" && len(path) > 0 && path[0] == "$" {
		path = path[1:]
	}
	p, ok := s.getProperty(path[0])
	if !ok {
		return nil, false
	}
	return p.getDescendent(path[1:])
}

func (s *Schema) getXMLDescendent(path []string) (*Schema, bool) {
	if len(path) == 0 {
		return s, true
	}
	p, ok := s.getProperty(path[0])
	if !ok {
		p, ok = s.getXMLChild(path[0])
		if !ok {
			return nil, false
		}
	}
	return p.getXMLDescendent(path[1:])
}

func (s *Schema) getDescendent(path []string) (*Schema, bool) {
	if len(path) == 0 {
		return s, true
	}
	if items, err := s.GetItems(); path[0] == "[*]" && err == nil {
		return items.getDescendent(path[1:])
	}
	p, ok := s.getProperty(path[0])
	if !ok {
		p, ok = s.getXMLChild(path[0])
		if !ok {
			return nil, false
		}
	}
	return p.getDescendent(path[1:])
}

func (s *Schema) GetItems() (*Schema, error) {
	if s.Items != nil && s.Items.Value != nil {
		itemsPathSplit := strings.Split(s.Items.Ref, "/")
		return NewSchema(s.Items.Value, itemsPathSplit[len(itemsPathSplit)-1]), nil
	}
	return nil, fmt.Errorf("no items present in schema with key = '%s'", s.key)
}

func (s *Schema) GetProperty(propertyKey string) (*Schema, error) {
	rv, ok := s.getProperty(propertyKey)
	if !ok {
		return nil, fmt.Errorf("failed to get property '%s'", propertyKey)
	}
	return rv, nil
}

func (s *Schema) getProperty(propertyKey string) (*Schema, bool) {
	var sc *openapi3.SchemaRef
	var ok bool
	if s.hasPolymorphicProperties() {
		polySchema := s.getFattnedPolymorphicSchema()
		sc, ok = polySchema.Properties[propertyKey]
	} else {
		sc, ok = s.Properties[propertyKey]
	}
	if !ok {
		return nil, false
	}
	return NewSchema(sc.Value, propertyKey), true
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
	return schema.getSelectListItems(key)
}

func (schema *Schema) getSelectListItems(key string) (*Schema, string) {
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

func (schema *Schema) GetSelectSchema(itemsKey, mediaType string) (*Schema, string, error) {
	if itemsKey == AnonymousColumnName {
		switch schema.Type {
		case "string", "integer":
			return schema, AnonymousColumnName, nil
		}
	}
	sc, str, err := schema.getSelectItemsSchema(itemsKey, mediaType)
	if err == nil {
		return sc, str, err
	}
	if schema != nil && schema.Properties != nil && len(schema.Properties) > 0 {
		return schema, "", nil
	}
	return nil, "", fmt.Errorf("unable to complete schema.GetSelectSchema() for schema = '%v' and itemsKey = '%s'", schema, itemsKey)
}

func (schema *Schema) getSelectItemsSchema(key string, mediaType string) (*Schema, string, error) {
	log.Infoln(fmt.Sprintf("schema.getSelectItemsSchema() key = '%s'", key))
	if key == "" {
		if schema.Items != nil && schema.Items.Value != nil {
			return NewSchema(schema.Items.Value, ""), "", nil
		}
		return schema, "", nil
	}
	switch mediaType {
	case media.MediaTypeXML, media.MediaTypeTextXML:
		pathResolver := openapitopath.NewXPathResolver()
		pathSplit := pathResolver.ToPathSlice(key)
		ss, ok := schema.getXMLDescendentInit(pathSplit)
		if ok && ss.Items != nil && ss.Items.Value != nil {
			rv, err := ss.GetItems()
			if rv.key == "" {
				for _, v := range rv.AllOf {
					if v.Ref != "" {
						rv.key = getPathSuffix(v.Ref)
						break
					}
				}
			}
			return rv, key, err
		}
		return nil, "", fmt.Errorf("could not resolve xml schema for key = '%s'", key)
	case media.MediaTypeJson, media.MediaTypeScimJson:
		if key != "" && strings.HasPrefix(key, "$") {
			pathResolver := openapitopath.NewJSONPathResolver()
			pathSplit := pathResolver.ToPathSlice(key)
			ss, ok := schema.getDescendentInit(pathSplit)
			if ok {
				return ss, key, nil
			}
			return nil, "", fmt.Errorf("could not resolve json schema for key = '%s'", key)
		}
		fallthrough
	default:
		return schema.deprecatedGetSelectItemsSchema(key, mediaType)
	}
}

func (schema *Schema) deprecatedGetSelectItemsSchema(key string, mediaType string) (*Schema, string, error) {
	var itemS *openapi3.Schema
	log.Infoln(fmt.Sprintf("schema.deprecatedGetSelectItemsSchema() key = '%s'", key))
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
		polySchema := schema.getFattnedPolymorphicSchema()
		if polySchema == nil {
			return nil, "", fmt.Errorf("polymorphic select reposnse parse failed")
		}
		return polySchema, "", nil
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

func (s *Schema) getXmlAlias() string {
	switch xml := s.XML.(type) {
	case map[string]interface{}:
		name, ok := xml["name"]
		if ok {
			switch name := name.(type) {
			case string:
				return name
			}
		}
	}
	for _, ao := range s.AllOf {
		if ao.Value != nil {
			aos := NewSchema(ao.Value, "")
			name := aos.getXmlAlias()
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func (s *Schema) getFatSchema(srs openapi3.SchemaRefs) *Schema {
	rv := NewSchema(s.Schema, s.key)
	if rv.Properties == nil {
		rv.Properties = make(openapi3.Schemas)
	}
	for k, val := range srs {
		log.Debugf("processing composite key number = %d, id = '%s'\n", k, val.Ref)
		ss := NewSchema(val.Value, "")
		if rv == nil {
			rv = ss
			continue
		}
		if ss.XML != nil {
			rv.XML = ss.XML
		}
		if ss.Type != "" {
			rv.Type = ss.Type
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
	sc := s.getFatSchema(srs)
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

func (s *Schema) isNotSimple() bool {
	switch s.Type {
	case "object", "array", "":
		return true
	default:
		return false
	}
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
	if s.hasPolymorphicProperties() {
		fs := s.getFattnedPolymorphicSchema()
		for k, v := range fs.Properties {
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
		return s.getFatSchema(s.AllOf)
	}
	if len(s.OneOf) > 0 {
		return s.getFatSchema(s.OneOf)
	}
	if len(s.AnyOf) > 0 {
		return s.getFatSchema(s.AnyOf)
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
	if s.Type == "object" || (s.hasPropertiesOrPolymorphicProperties() && s.isNotSimple()) {
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
			// TODO: prevent endless recursion
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

func (s *Schema) unmarshalXMLResponseBody(body io.ReadCloser, path string) (interface{}, *xmlquery.Node, error) {
	return xmlmap.GetSubObjTyped(body, path, s.Schema)
}

func (s *Schema) unmarshalJSONResponseBody(body io.ReadCloser, path string) (interface{}, interface{}, error) {
	var target interface{}
	err := json.NewDecoder(body).Decode(&target)
	if err != nil {
		return nil, nil, err
	}
	processedResponse, err := jsonpath.Get(path, target)
	if err != nil {
		return nil, nil, err
	}
	return processedResponse, target, nil
}

func (s *Schema) unmarshalResponse(r *http.Response) (interface{}, error) {
	body := r.Body
	if body != nil {
		defer body.Close()
	} else {
		return nil, nil
	}
	var target interface{}
	mediaType, err := media.GetResponseMediaType(r)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case media.MediaTypeJson, media.MediaTypeScimJson:
		err = json.NewDecoder(body).Decode(&target)
	case media.MediaTypeXML, media.MediaTypeTextXML:
		return nil, fmt.Errorf("xml disallowed here")
	case media.MediaTypeOctetStream:
		target, err = io.ReadAll(body)
	case media.MediaTypeTextPlain, media.MediaTypeHTML:
		var b []byte
		b, err = io.ReadAll(body)
		if err == nil {
			target = string(b)
		}
	default:
		target, err = io.ReadAll(body)
	}
	return target, err
}

func (s *Schema) unmarshalResponseAtPath(r *http.Response, path string) (*response.Response, error) {

	mediaType, err := media.GetResponseMediaType(r)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case media.MediaTypeXML, media.MediaTypeTextXML:
		pathResolver := openapitopath.NewXPathResolver()
		pathSplit := pathResolver.ToPathSlice(path)
		ss, ok := s.getXMLDescendentInit(pathSplit)
		if !ok {
			return nil, fmt.Errorf("cannot find xml descendent for path %+v", pathSplit)
		}
		processedResponse, rawResponse, err := ss.unmarshalXMLResponseBody(r.Body, path)
		if err != nil {
			return nil, err
		}
		return response.NewResponse(processedResponse, rawResponse, r), nil
	case media.MediaTypeJson, media.MediaTypeScimJson:
		// TODO: follow same pattern as XML, but with json path
		if path != "" && strings.HasPrefix(path, "$") {
			pathResolver := openapitopath.NewJSONPathResolver()
			pathSplit := pathResolver.ToPathSlice(path)
			ss, ok := s.getDescendentInit(pathSplit)
			if !ok {
				return nil, fmt.Errorf("cannot find json descendent for path %+v", pathSplit)
			}
			processedResponse, rawResponse, err := ss.unmarshalJSONResponseBody(r.Body, path)
			if err != nil {
				return nil, err
			}
			return response.NewResponse(processedResponse, rawResponse, r), nil
		}
		fallthrough
	default:
		processedResponse, err := s.unmarshalResponse(r)
		if err != nil {
			return nil, err
		}
		return response.NewResponse(processedResponse, processedResponse, r), nil
	}
}

func (s *Schema) ProcessHttpResponse(r *http.Response, path string) (*response.Response, error) {
	defer r.Body.Close()
	target, err := s.unmarshalResponseAtPath(r, path)
	if err == nil && r.StatusCode >= 400 {
		err = fmt.Errorf(fmt.Sprintf("HTTP response error.  Status code %d.  Detail: %s", r.StatusCode, string(util.InterfaceToBytes(target, true))))
	}
	if err == io.EOF {
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			boilerplate := map[string]interface{}{"result": "The Operation Completed Successfully"}
			return response.NewResponse(boilerplate, boilerplate, r), nil
		}
	}
	switch rv := target.GetProcessedBody().(type) {
	case string, int:
		boilerplate := map[string]interface{}{AnonymousColumnName: []interface{}{rv}}
		return response.NewResponse(boilerplate, target.GetBody(), target.GetHttpResponse()), nil
	}
	return target, err
}

func (s *Schema) DeprecatedProcessHttpResponse(response *http.Response, path string) (map[string]interface{}, error) {
	target, err := s.ProcessHttpResponse(response, path)
	if err != nil {
		return nil, err
	}
	switch rv := target.GetProcessedBody().(type) {
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
