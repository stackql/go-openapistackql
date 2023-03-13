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
	_                   Schema = (*standardSchema)(nil)
)

type Schema interface {
	ConditionIsValid(lhs string, rhs interface{}) bool
	DeprecatedProcessHttpResponse(response *http.Response, path string) (map[string]interface{}, error)
	FindByPath(path string, visited map[string]bool) Schema
	GetAllColumns() []string
	GetItemProperty(k string) (Schema, bool)
	GetItems() (Schema, error)
	GetName() string
	GetPath() string
	GetProperties() (Schemas, error)
	GetProperty(propertyKey string) (Schema, error)
	GetSelectionName() string
	GetType() string
	IsArrayRef() bool
	IsBoolean() bool
	IsFloat() bool
	IsIntegral() bool
	IsRequired(key string) bool
	ProcessHttpResponseTesting(r *http.Response, path string, defaultMediaType string) (*response.Response, error)
	ToDescriptionMap(extended bool) map[string]interface{}
	// not exported, but essential
	deprecatedGetSelectItemsSchema(key string, mediaType string) (Schema, string, error)
	getAllOf() openapi3.SchemaRefs
	getDescendent(path []string) (Schema, bool)
	getFatItemsSchema(srs openapi3.SchemaRefs) Schema
	getItemsRef() (*openapi3.SchemaRef, bool)
	getKey() string
	getOpenapiSchema() (*openapi3.Schema, bool)
	getPropertiesColumns() []ColumnDescriptor
	getXmlAlias() string
	getXMLChild(path string, isTerminal bool) (Schema, bool)
	getXMLDescendent(path []string) (Schema, bool)
	isXmlWrapped() bool
	setKey(string)
	unmarshalJSONResponseBody(body io.ReadCloser, path string) (interface{}, interface{}, error)
	unmarshalXMLResponseBody(body io.ReadCloser, path string) (interface{}, *xmlquery.Node, error)
}

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

func (s *standardSchema) GetItemProperty(k string) (Schema, bool) {
	raw, ok := s.Items.Value.Properties[k]
	if !ok {
		return nil, false
	}
	return NewSchema(raw.Value, s.svc, k, ""), ok
}

func (s *standardSchema) getItemsRef() (*openapi3.SchemaRef, bool) {
	if s.Items != nil {
		return s.Items, true
	}
	return nil, false
}

func (s *standardSchema) getKey() string {
	return s.key
}

func (s *standardSchema) GetType() string {
	return s.Type
}

func (s *standardSchema) setKey(k string) {
	s.key = k
}

func (s *standardSchema) ConditionIsValid(lhs string, rhs interface{}) bool {
	return providerTypeConditionIsValid(s.Type, lhs, rhs)
}

func (s *standardSchema) getAllOf() openapi3.SchemaRefs {
	return s.AllOf
}

func (s *standardSchema) getOpenapiSchema() (*openapi3.Schema, bool) {
	return s.Schema, s.Schema != nil
}

type standardSchema struct {
	*openapi3.Schema
	svc            *Service
	key            string
	alwaysRequired bool
	path           string
}

func copyOpenapiSchema(inSchema *openapi3.Schema) *openapi3.Schema {
	properties := make(openapi3.Schemas)
	for k, v := range inSchema.Properties {
		properties[k] = v
	}
	rv := openapi3.NewSchema()
	rv.Properties = properties

	rv.Items = inSchema.Items

	rv.ExtensionProps = inSchema.ExtensionProps
	rv.OneOf = inSchema.OneOf
	rv.AnyOf = inSchema.AnyOf
	rv.AllOf = inSchema.AllOf
	rv.Not = inSchema.Not
	rv.Type = inSchema.Type
	rv.Title = inSchema.Title
	rv.Format = inSchema.Format
	rv.Description = inSchema.Description
	rv.Enum = inSchema.Enum
	rv.Default = inSchema.Default
	rv.Example = inSchema.Example
	rv.ExternalDocs = inSchema.ExternalDocs
	rv.UniqueItems = inSchema.UniqueItems
	rv.ExclusiveMin = inSchema.ExclusiveMin
	rv.ExclusiveMax = inSchema.ExclusiveMax
	rv.Nullable = inSchema.Nullable
	rv.ReadOnly = inSchema.ReadOnly
	rv.WriteOnly = inSchema.WriteOnly
	rv.AllowEmptyValue = inSchema.AllowEmptyValue
	rv.XML = inSchema.XML
	rv.Deprecated = inSchema.Deprecated
	rv.Min = inSchema.Min
	rv.Max = inSchema.Max
	rv.MultipleOf = inSchema.MultipleOf
	rv.MinLength = inSchema.MinLength
	rv.MaxLength = inSchema.MaxLength
	rv.Pattern = inSchema.Pattern
	rv.MinItems = inSchema.MinItems
	rv.MaxItems = inSchema.MaxItems
	rv.Required = inSchema.Required
	rv.MinProps = inSchema.MinProps
	rv.MaxProps = inSchema.MaxProps
	rv.AdditionalPropertiesAllowed = inSchema.AdditionalPropertiesAllowed
	rv.AdditionalProperties = inSchema.AdditionalProperties
	rv.Discriminator = inSchema.Discriminator

	return rv
}

type Schemas map[string]Schema

func NewSchema(sc *openapi3.Schema, svc *Service, key string, path string) *standardSchema {
	return newSchema(sc, svc, key, path)
}

func (sc *standardSchema) GetPath() string {
	return sc.path
}

func newSchema(sc *openapi3.Schema, svc *Service, key string, path string) *standardSchema {
	var alwaysRequired bool
	if sc.Extensions != nil {
		if ar, ok := sc.Extensions[ExtensionKeyAlwaysRequired]; ok {
			if pr, ok := ar.(bool); ok && pr {
				alwaysRequired = true
			}
		}
	}
	return &standardSchema{
		Schema:         sc,
		svc:            svc,
		key:            key,
		alwaysRequired: alwaysRequired,
		path:           path,
	}
}

func (s *standardSchema) isObjectSchemaImplicitlyUnioned() bool {
	if s.svc == nil {
		return false
	}
	return s.svc.isObjectSchemaImplicitlyUnioned()
}

func (s *standardSchema) GetProperties() (Schemas, error) {
	return s.getProperties(), nil
}

func (s *standardSchema) getProperties() Schemas {
	retVal := make(Schemas)
	if s.isObjectSchemaImplicitlyUnioned() {
		return s.getInplicitlyUnionedProperties()
	}
	if s.hasPolymorphicProperties() && len(s.Properties) == 0 {
		ss := s.getFattnedPolymorphicSchema()
		if ss != nil {
			for k, sr := range ss.Properties {
				retVal[k] = NewSchema(sr.Value, s.svc, k, sr.Ref)
			}
		}
	}
	for k, sr := range s.Properties {
		retVal[k] = NewSchema(sr.Value, s.svc, k, sr.Ref)
	}
	return retVal
}

// This is a horrendous hack to cover weird `properties` + `allOf` seen
// all across azure autorest docs.  It is opt-in via config and
// should, nay must, be removed when time permits
func (s *standardSchema) getInplicitlyUnionedProperties() Schemas {
	retVal := make(Schemas)
	if s.hasPolymorphicProperties() {
		ss := s.getFattnedPolymorphicSchema()
		if ss != nil {
			for k, sr := range ss.Properties {
				retVal[k] = NewSchema(sr.Value, s.svc, k, sr.Ref)
			}
		}
	}
	for k, sr := range s.Properties {
		retVal[k] = NewSchema(sr.Value, s.svc, k, sr.Ref)
	}
	return retVal
}

func getPathSuffix(path string) string {
	pathSplit := strings.Split(path, "/")
	return pathSplit[len(pathSplit)-1]
}

func (s *standardSchema) GetName() string {
	return s.getName()
}

func (s *standardSchema) GetSelectionName() string {
	if s.Items != nil {
		return getPathSuffix(s.Items.Ref)
	}
	return s.getName()
}

func (s *standardSchema) getName() string {
	return getPathSuffix(s.key)
}

func (s *standardSchema) getXMLALiasOrName() string {
	xa := s.getXmlAlias()
	if xa != "" {
		return xa
	}
	return s.getName()
}

func (s *standardSchema) IsRequired(key string) bool {
	for _, req := range s.Required {
		if req == key {
			return true
		}
	}
	return false
}

func (s *standardSchema) getXMLChild(path string, isTerminal bool) (Schema, bool) {
	xmlAlias := s.getXmlAlias()
	if xmlAlias == path {
		return s, true
	}
	for _, v := range s.getProperties() {
		if v.getXmlAlias() == path {
			return v, true
		}
	}
	if s.Type == "array" && s.Items != nil && s.Items.Value != nil {
		ss := NewSchema(s.Items.Value, s.svc, "", s.Items.Ref)
		ds, ok := ss.getXMLChild(path, isTerminal)
		if ok {
			if !isTerminal {
				return ds, true
			}
			return s, true
		}
		return nil, false
	}
	for _, v := range s.AllOf {
		if v.Value == nil {
			continue
		}
		si := v.Value
		if si.Type == "array" && si.Items != nil && si.Items.Value != nil {
			ss := NewSchema(si.Items.Value, s.svc, "", si.Items.Ref)
			ds, ok := ss.getXMLChild(path, isTerminal)
			if ok {
				if !isTerminal {
					return ds, true
				}
				return NewSchema(si, s.svc, getPathSuffix(si.Items.Ref), si.Items.Ref), true
			}
			return nil, false
		}
	}
	return nil, false
}

func (s *standardSchema) getXMLDescendentInit(path []string) (Schema, bool) {
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
		p, ok = s.getXMLChild(path[0], len(path) <= 1)
		if !ok {
			return nil, false
		}
	}
	return p.getXMLDescendent(path[1:])
}

func (s *standardSchema) getDescendentInit(path []string) (Schema, bool) {
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

func (s *standardSchema) getXmlAttribute(key string) (interface{}, bool) {
	if s.XML != nil {
		if xmlMap, ok := s.XML.(map[string]interface{}); ok {
			rv, ok := xmlMap[key]
			return rv, ok
		}
	}
	return nil, false
}

func (s *standardSchema) getXmlName() (string, bool) {
	if name, ok := s.getXmlAttribute("name"); ok {
		nameStr, ok := name.(string)
		return nameStr, ok
	}
	if len(s.AllOf) > 0 {
		for _, ss := range s.AllOf {
			if ss.Value == nil {
				continue
			}
			ns := newSchema(ss.Value, s.svc, "", ss.Ref)
			if sn, ok := ns.getXmlName(); ok {
				return sn, true
			}
		}
	}
	return "", false
}

func (s *standardSchema) isItemsXmlWrapped() bool {
	if s.Items != nil && s.Items.Value == nil {
		itemsSchema := newSchema(s.Items.Value, s.svc, "", s.Items.Ref)
		return itemsSchema.isXmlWrapped()
	}
	if len(s.AllOf) > 0 {
		fs := s.getFatItemsSchema(s.AllOf)
		return fs.isXmlWrapped()
	}
	return false
}

func (s *standardSchema) isXmlWrapped() bool {
	// This is a hack until aws.ec2 is fixed
	if _, ok := s.getXmlName(); ok {
		return true
	}
	wrapped, ok := s.getXmlAttribute("wrapped")
	if !ok {
		return false
	}
	wrappedBool, isBool := wrapped.(bool)
	if len(s.AllOf) > 0 {
		for _, ss := range s.AllOf {
			if ss.Value == nil {
				continue
			}
			ns := newSchema(ss.Value, s.svc, "", ss.Ref)
			if ns.isXmlWrapped() {
				return true
			}
		}
	}
	return isBool && wrappedBool
}

func (s *standardSchema) getXMLTerminal() (Schema, bool) {
	if !s.hasPolymorphicProperties() {
		return s, true
	}
	rv := s.getFattnedPolymorphicSchema()
	if rv.Type == "array" && !s.isItemsXmlWrapped() {
		items, err := rv.GetItems()
		if err != nil {
			return nil, false
		}
		return items, true
	}
	return rv, true
}

func (s *standardSchema) getXMLDescendent(path []string) (Schema, bool) {
	if len(path) == 0 {
		return s.getXMLTerminal()
	}
	if len(path) == 1 && path[0] == "*" {
		return s.getXMLTerminal()
	}
	p, ok := s.getProperty(path[0])
	if !ok {
		p, ok = s.getXMLChild(path[0], len(path) <= 1)
		if !ok {
			return nil, false
		}
	}
	return p.getXMLDescendent(path[1:])
}

func (s *standardSchema) getDescendent(path []string) (Schema, bool) {
	if len(path) == 0 {
		return s, true
	}
	if items, err := s.GetItems(); path[0] == "[*]" && err == nil {
		return items.getDescendent(path[1:])
	}
	p, ok := s.getProperty(path[0])
	if !ok {
		p, ok = s.getXMLChild(path[0], len(path) <= 1)
		if !ok {
			return nil, false
		}
	}
	return p.getDescendent(path[1:])
}

func (s *standardSchema) GetItems() (Schema, error) {
	if len(s.AllOf) > 0 {
		ns := s.getFatItemsSchema(s.getAllOf())
		switch ns := ns.(type) {
		case *standardSchema:
			s = ns
		default:
			return nil, fmt.Errorf("failed to get items for schema with type = '%T'", ns)
		}
	}
	if s.Items != nil && s.Items.Value != nil {
		itemsPathSplit := strings.Split(s.Items.Ref, "/")
		return NewSchema(s.Items.Value, s.svc, itemsPathSplit[len(itemsPathSplit)-1], s.Items.Ref), nil
	}
	return nil, fmt.Errorf("no items present in schema with key = '%s'", s.key)
}

func (s *standardSchema) GetProperty(propertyKey string) (Schema, error) {
	rv, ok := s.getProperty(propertyKey)
	if !ok {
		return nil, fmt.Errorf("failed to get property '%s'", propertyKey)
	}
	return rv, nil
}

func (s *standardSchema) getProperty(propertyKey string) (Schema, bool) {
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
	return NewSchema(sc.Value, s.svc, getPathSuffix(sc.Ref), sc.Ref), true
}

func (s *standardSchema) IsIntegral() bool {
	return s.Type == "int" || s.Type == "integer"
}

func (s *standardSchema) IsBoolean() bool {
	return s.Type == "bool" || s.Type == "boolean"
}

func (s *standardSchema) IsFloat() bool {
	return s.Type == "float" || s.Type == "float64"
}

func (sc *standardSchema) GetPropertySchema(key string) (*standardSchema, error) {
	absentErr := fmt.Errorf("property schema not present for key '%s'", key)
	sh, ok := sc.Properties[key]
	if !ok {
		return nil, absentErr
	}
	return NewSchema(
		sh.Value,
		sc.svc,
		key,
		sh.Ref,
	), nil
}

func (sc *standardSchema) GetItemsSchema() (Schema, error) {
	absentErr := fmt.Errorf("items schema not present")
	sh := sc.Items
	if sh.Value != nil {
		return NewSchema(
			sh.Value,
			sc.svc,
			"",
			sh.Ref,
		), nil
	}
	return nil, absentErr
}

func (schema *standardSchema) GetSelectListItems(key string) (Schema, string) {
	return schema.getSelectListItems(key)
}

func (schema *standardSchema) getSelectListItems(key string) (Schema, string) {
	propS, ok := schema.Properties[key]
	if !ok {
		return nil, ""
	}
	itemS := propS.Value
	if itemS != nil {
		return NewSchema(
			itemS,
			schema.svc,
			"",
			propS.Ref,
		), key
	}
	return nil, ""
}

func (schema *standardSchema) GetSelectSchema(itemsKey, mediaType string) (Schema, string, error) {
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

// TODO: implement upwards-searchable configurable type set matching
func (schema *standardSchema) extractMediaTypeSynonym(mediaType string) string {
	m, ok := media.DefaultMediaFuzzyMatcher.Find(mediaType)
	if ok {
		return m
	}
	return mediaType
}

func (schema *standardSchema) getSelectItemsSchema(key string, mediaType string) (Schema, string, error) {
	log.Infoln(fmt.Sprintf("schema.getSelectItemsSchema() key = '%s'", key))
	if key == "" {
		if schema.Items != nil && schema.Items.Value != nil {
			return NewSchema(schema.Items.Value, schema.svc, "", schema.Items.Ref), "", nil
		}
		return schema, "", nil
	}
	switch schema.extractMediaTypeSynonym(mediaType) {
	case media.MediaTypeXML:
		pathResolver := openapitopath.NewXPathResolver()
		pathSplit := pathResolver.ToPathSlice(key)
		ss, ok := schema.getXMLDescendentInit(pathSplit)
		if ok {
			_, itemsRefExists := ss.getItemsRef()
			if itemsRefExists {
				rv, err := ss.GetItems()
				if rv.getKey() == "" {
					for _, v := range rv.getAllOf() {
						if v.Ref != "" {
							rv.setKey(getPathSuffix(v.Ref))
							break
						}
					}
				}
				return rv, key, err
			}
		}
		if ok {
			return ss, key, nil
		}
		return nil, "", fmt.Errorf("could not resolve xml schema for key = '%s'", key)
	case media.MediaTypeJson:
		if key != "" && strings.HasPrefix(key, "$") {
			pathResolver := openapitopath.NewJSONPathResolver()
			pathSplit := pathResolver.ToPathSlice(key)
			ss, ok := schema.getDescendentInit(pathSplit)
			if ok {
				_, itemsRefExists := ss.getItemsRef()
				if itemsRefExists {
					rv, err := ss.GetItems()
					if rv.getKey() == "" {
						for _, v := range rv.getAllOf() {
							if v.Ref != "" {
								rv.setKey(getPathSuffix(v.Ref))
								break
							}
						}
					}
					return rv, key, err
				}
			}
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

func (schema *standardSchema) deprecatedGetSelectItemsSchema(key string, mediaType string) (Schema, string, error) {
	var itemS *openapi3.Schema
	var schemaPath string
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
		schemaPath = propS.Ref
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
			schema.svc,
			key,
			schemaPath,
		)
		rv, err := s.GetItems()
		return rv, key, err
	}
	return nil, "", fmt.Errorf("could not find items for key = '%s'", key)
}

func (s *standardSchema) getType() string {
	if s.Type != "" {
		return s.Type
	}
	for _, sRef := range s.AllOf {
		if sRef != nil && sRef.Value != nil && sRef.Value.Type != "" {
			return sRef.Value.Type
		}
	}
	return ""
}

func (s *standardSchema) getTitle() string {
	if s.Title != "" {
		return s.Title
	}
	for _, sRef := range s.AllOf {
		if sRef != nil && sRef.Value != nil && sRef.Value.Title != "" {
			return sRef.Value.Title
		}
	}
	return ""
}

func (s *standardSchema) getDescription() string {
	if s.Description != "" {
		return s.Description
	}
	for _, sRef := range s.AllOf {
		if sRef != nil && sRef.Value != nil && sRef.Value.Description != "" {
			return sRef.Value.Description
		}
	}
	return ""
}

func (s *standardSchema) toFlatDescriptionMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	retVal["name"] = s.getTitle()
	retVal["type"] = s.getType()
	if extended {
		retVal["description"] = s.getDescription()
	}
	return retVal
}

func (s *standardSchema) GetAllColumns() []string {
	log.Infoln(fmt.Sprintf("s = %v", *s))
	var retVal []string
	properties := s.getProperties()
	if s.Type == "object" || (len(properties) > 0) {
		for k, val := range properties {
			_, valSchemaExists := val.getOpenapiSchema()
			if valSchemaExists {
				retVal = append(retVal, k)
			}
		}
	} else if s.Type == "array" {
		if items := s.Items.Value; items != nil {
			iS := NewSchema(items, s.svc, "", s.Items.Ref)
			return iS.GetAllColumns()
		}
	}
	switch s.Type {
	case "string", "bool", "integer":
		return []string{AnonymousColumnName}
	}
	return retVal
}

func (s *standardSchema) IsArrayRef() bool {
	return s.Items != nil && s.Items.Value != nil
}

func (s *standardSchema) getPropertiesColumns() []ColumnDescriptor {
	var cols []ColumnDescriptor
	for k, val := range s.Properties {
		valSchema := val.Value
		if valSchema != nil {
			col := newColumnDescriptor(
				"",
				k,
				"",
				"",
				nil,
				NewSchema(
					valSchema,
					s.svc,
					k,
					val.Ref,
				),
				nil,
			)
			cols = append(cols, col)
		}
	}
	return cols
}

func (s *standardSchema) getAllOfColumns() []ColumnDescriptor {
	return s.getAllSchemaRefsColumns(s.AllOf)
}

func (s *standardSchema) getAnyOfColumns() []ColumnDescriptor {
	return s.getAllSchemaRefsColumns(s.AnyOf)
}

func (s *standardSchema) getOneOfColumns() []ColumnDescriptor {
	return s.getAllSchemaRefsColumns(s.OneOf)
}

func getSchemaName(sr *openapi3.SchemaRef) string {
	spl := strings.Split(sr.Ref, "/")
	if l := len(spl); l > 0 {
		return spl[l-1]
	}
	return ""
}

func (s *standardSchema) getXmlAlias() string {
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
			aos := NewSchema(ao.Value, s.svc, "", ao.Ref)
			name := aos.getXmlAlias()
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func (s *standardSchema) getFatSchema(srs openapi3.SchemaRefs) *standardSchema {
	var copiedSchema *openapi3.Schema
	if s.Schema != nil {
		copiedSchema = copyOpenapiSchema(s.Schema)
	}
	rv := newSchema(copiedSchema, s.svc, s.key, s.path)
	if rv.Properties == nil {
		rv.Properties = make(openapi3.Schemas)
	}
	for k, val := range srs {
		log.Debugf("processing composite key number = %d, id = '%s'\n", k, val.Ref)
		ss := newSchema(val.Value, s.svc, getPathSuffix(val.Ref), val.Ref)
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

func (s *standardSchema) getFatItemsSchema(srs openapi3.SchemaRefs) Schema {
	copySchema := copyOpenapiSchema(s.Schema)
	rv := newSchema(copySchema, s.svc, s.key, s.path)
	if rv.Properties == nil {
		rv.Properties = make(openapi3.Schemas)
	}
	for k, val := range srs {
		log.Debugf("processing composite key number = %d, id = '%s'\n", k, val.Ref)
		ss := newSchema(val.Value, s.svc, getPathSuffix(val.Ref), val.Ref)
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
		if ss.Items != nil {
			rv.Items = ss.Items
		}
	}
	return rv
}

func (s *standardSchema) getFatSchemaWithOverwrites(srs openapi3.SchemaRefs) Schema {
	var copiedSchema *openapi3.Schema
	if s.Schema != nil {
		copiedSchema = copyOpenapiSchema(s.Schema)
	}
	rv := newSchema(copiedSchema, s.svc, s.key, s.path)
	if rv.Properties == nil {
		rv.Properties = make(openapi3.Schemas)
	}
	for k, val := range srs {
		log.Debugf("processing composite key number = %d, id = '%s'\n", k, val.Ref)
		ss := newSchema(val.Value, s.svc, "", val.Ref)
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
				continue
			}
			rv.Properties[k] = sRef
		}
	}
	return rv
}

func (s *standardSchema) getAllSchemaRefsColumns(srs openapi3.SchemaRefs) []ColumnDescriptor {
	sc := s.getFatSchema(srs)
	st := sc.Tabulate(false)
	return st.GetColumns()
}

func (s *standardSchema) getAllSchemaRefsColumnsShallow(srs openapi3.SchemaRefs) []ColumnDescriptor {
	sc := s.getFatSchemaWithOverwrites(srs)
	return sc.getPropertiesColumns()
}

func (s *standardSchema) hasPolymorphicProperties() bool {
	if len(s.AllOf) > 0 || len(s.AnyOf) > 0 || len(s.OneOf) > 0 {
		return true
	}
	return false
}

func (s *standardSchema) hasPropertiesOrPolymorphicProperties() bool {
	if s.Properties != nil && len(s.Properties) > 0 {
		return true
	}
	return s.hasPolymorphicProperties()
}

func (s *standardSchema) isNotSimple() bool {
	switch s.Type {
	case "object", "array", "":
		return true
	default:
		return false
	}
}

func (s *standardSchema) Tabulate(omitColumns bool) *Tabulation {
	if s.Type == "object" || (s.hasPropertiesOrPolymorphicProperties() && s.Type != "array") {
		var cols []ColumnDescriptor
		if !omitColumns {
			if s.isObjectSchemaImplicitlyUnioned() {
				keysUsed := make(map[string]struct{})
				cols = s.getPropertiesColumns()
				for _, col := range cols {
					keysUsed[col.Name] = struct{}{}
				}
				var additionalCols []ColumnDescriptor
				if len(s.AllOf) > 0 {
					additionalCols = s.getAllSchemaRefsColumnsShallow(s.AllOf)
				}
				for _, col := range additionalCols {
					if _, ok := keysUsed[col.Name]; !ok {
						cols = append(cols, col)
						keysUsed[col.Name] = struct{}{}
					}
				}
			} else if len(s.Properties) > 0 {
				cols = s.getPropertiesColumns()
			} else if len(s.AllOf) > 0 {
				cols = s.getAllOfColumns()
			} else if len(s.AnyOf) > 0 {
				cols = s.getAnyOfColumns()
			} else if len(s.OneOf) > 0 {
				cols = s.getOneOfColumns()
			}
		}
		return &Tabulation{columns: cols, name: s.GetName(), schema: s}
	} else if s.Type == "array" {
		if items := s.Items.Value; items != nil {
			rv := newSchema(items, s.svc, "", s.Items.Ref).Tabulate(omitColumns)
			return rv
		}
	} else if s.Type == "string" {
		cd := newColumnDescriptor("", AnonymousColumnName, "", "", nil, s, nil)
		if omitColumns {
			return &Tabulation{columns: []ColumnDescriptor{}, name: s.Title, schema: s}
		}
		return &Tabulation{columns: []ColumnDescriptor{cd}, name: s.Title, schema: s}
	}
	return nil
}

func (s *standardSchema) ToDescriptionMap(extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	if s.Type == "array" {
		items := s.Items.Value
		if items != nil {
			return NewSchema(items, s.svc, "", s.Items.Ref).ToDescriptionMap(extended)
		}
	}
	// TODO:
	//     - Ensure this logic conforms to openapi3 doc rules.
	//     - Add integration testing to ensure same, corner cases.
	if s.Type == "object" {
		for k, v := range s.Properties {
			p := v.Value
			if p != nil {
				pm := NewSchema(p, s.svc, "", v.Ref).toFlatDescriptionMap(extended)
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
				pm := NewSchema(p, s.svc, "", v.Ref).toFlatDescriptionMap(extended)
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

func (s *standardSchema) getFattnedPolymorphicSchema() *standardSchema {
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

func (s *standardSchema) FindByPath(path string, visited map[string]bool) Schema {
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
				return NewSchema(rv, s.svc, k, v.Ref)
			}
			ss := NewSchema(v.Value, s.svc, k, v.Ref)
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

func (s *standardSchema) unmarshalXMLResponseBody(body io.ReadCloser, path string) (interface{}, *xmlquery.Node, error) {
	return xmlmap.GetSubObjTyped(body, path, s.Schema)
}

func (s *standardSchema) unmarshalJSONResponseBody(body io.ReadCloser, path string) (interface{}, interface{}, error) {
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

func (s *standardSchema) unmarshalResponse(r *http.Response) (interface{}, error) {
	body := r.Body
	if body != nil {
		defer body.Close()
	} else {
		return nil, nil
	}
	var target interface{}
	mediaType, err := media.GetResponseMediaType(r, "")
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

func (s *standardSchema) unmarshalResponseAtPath(r *http.Response, path string, defaultMediaType string) (*response.Response, error) {

	mediaType, err := media.GetResponseMediaType(r, defaultMediaType)
	if err != nil {
		return nil, err
	}
	switch s.extractMediaTypeSynonym(mediaType) {
	case media.MediaTypeXML:
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
	case media.MediaTypeJson:
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

func (s *standardSchema) ProcessHttpResponseTesting(r *http.Response, path string, defaultMediaType string) (*response.Response, error) {
	return s.processHttpResponse(r, path, defaultMediaType)
}

func (s *standardSchema) processHttpResponse(r *http.Response, path string, defaultMediaType string) (*response.Response, error) {
	defer r.Body.Close()
	target, err := s.unmarshalResponseAtPath(r, path, defaultMediaType)
	if err == nil && r.StatusCode >= 400 {
		err = fmt.Errorf(fmt.Sprintf("HTTP response error.  Status code %d.  Detail: %s", r.StatusCode, string(util.InterfaceToBytes(target, true))))
	}
	if err == io.EOF {
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			boilerplate := map[string]interface{}{"result": "The Operation Completed Successfully"}
			return response.NewResponse(boilerplate, boilerplate, r), nil
		}
	}
	if target == nil || target.GetProcessedBody() == nil {
		return target, err
	}
	switch rv := target.GetProcessedBody().(type) {
	case string, int:
		boilerplate := map[string]interface{}{AnonymousColumnName: []interface{}{rv}}
		return response.NewResponse(boilerplate, target.GetBody(), target.GetHttpResponse()), nil
	}
	return target, err
}

func (s *standardSchema) DeprecatedProcessHttpResponse(response *http.Response, path string) (map[string]interface{}, error) {
	target, err := s.processHttpResponse(response, path, "")
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
