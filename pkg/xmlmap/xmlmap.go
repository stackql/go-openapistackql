package xmlmap

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/antchfx/xmlquery"
)

/*
This package is all about dealing with the difficulty in XML deserialization.

The problem is stated in [the golang xml package](https://pkg.go.dev/encoding/xml#pkg-note-BUG) as:

> Mapping between XML elements and data structures is inherently flawed: an XML element is an order-dependent collection of anonymous values, while a data structure is an order-independent collection of named values. See package json for a textual representation more suitable to data structures.

*/

type kv struct {
	k, v   string
	isNull bool
}

func getNodeKeyVal(node *xmlquery.Node) (kv, error) {
	switch node.Type {
	case xmlquery.TextNode, xmlquery.CharDataNode, xmlquery.CommentNode:
		ts := strings.TrimSpace(node.Data)
		if ts == "" {
			return kv{isNull: true}, nil
		}
		return kv{}, fmt.Errorf("cannot get kv for node")
	default:
		return kv{
			k: node.Data,
			v: node.InnerText(),
		}, nil
	}
}

func getNodeMap(node *xmlquery.Node) (map[string]string, error) {
	rv := make(map[string]string)
	switch node.Type {
	case xmlquery.TextNode, xmlquery.CharDataNode, xmlquery.CommentNode:
		return nil, nil
	default:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			kv, err := getNodeKeyVal(child)
			if err != nil {
				return nil, err
			}
			if kv.isNull {
				continue
			}
			rv[kv.k] = kv.v
		}
	}
	return rv, nil
}

func xmlNameFromRefs(refs openapi3.SchemaRefs) (string, bool) {
	for _, sRef := range refs {
		if sRef != nil || sRef.Value != nil {
			p, ok := xmlNameFromSchema(sRef.Value)
			if ok {
				return p, true
			}
		}
	}
	return "", false
}

func xmlNameFromSchema(schema *openapi3.Schema) (string, bool) {
	switch xml := schema.XML.(type) {
	case map[string]interface{}:
		name, ok := xml["name"]
		if ok {
			switch name := name.(type) {
			case string:
				return name, true
			}
		}
	}
	if len(schema.AllOf) > 0 {
		rv, ok := xmlNameFromRefs(schema.AllOf)
		if ok {
			return rv, true
		}
	}
	return "", false
}

func getPropertyByXMLAnnotation(schema *openapi3.Schema, name string) (*openapi3.Schema, bool) {
	for _, v := range schema.Properties {
		if v != nil && v.Value != nil {
			xmlName, ok := xmlNameFromSchema(v.Value)
			if ok && xmlName == name {
				return v.Value, true
			}
		}
	}
	return nil, false
}

func castXMLValue(inVal string, schema *openapi3.Schema) (interface{}, error) {
	ty, _ := getTypeFromSchema(schema)
	if ty == "" {
		if len(schema.AllOf) > 0 {
			t, ok := getTypeFromRefs(schema.AllOf)
			if ok {
				ty = t
			}
		}
	}
	switch ty {
	case "object", "array", "string":
		return inVal, nil
	case "integer", "int64":
		return strconv.Atoi(inVal)
	case "bool":
		return strings.ToLower(inVal) == "true", nil
	default:
		return inVal, nil
	}
}

func getPropertyFromRefs(refs openapi3.SchemaRefs, key string) (*openapi3.Schema, bool) {
	for _, sRef := range refs {
		if sRef != nil || sRef.Value != nil {
			p, ok := getPropertyFromSchema(sRef.Value, key)
			if ok {
				return p, true
			}
		}
	}
	return nil, false
}

func getTypeFromSchema(schema *openapi3.Schema) (string, bool) {
	if schema.Type != "" {
		return schema.Type, true
	}
	if len(schema.AllOf) > 0 {
		t, ok := getTypeFromRefs(schema.AllOf)
		if ok {
			return t, true
		}
	}
	return "", false
}

func getTypeFromRefs(refs openapi3.SchemaRefs) (string, bool) {
	for _, sRef := range refs {
		if sRef != nil || sRef.Value != nil {
			t, ok := getTypeFromSchema(sRef.Value)
			if ok {
				return t, true
			}
		}
	}
	return "", false
}

func getPropertyFromSchema(schema *openapi3.Schema, key string) (*openapi3.Schema, bool) {
	ref, ok := schema.Properties[key]
	if ok {
		return ref.Value, true
	}
	s, ok := getPropertyByXMLAnnotation(schema, key)
	if ok {
		return s, true
	}
	if len(schema.AllOf) > 0 {
		p, ok := getPropertyFromRefs(schema.AllOf, key)
		if ok {
			return p, true
		}
	}
	return nil, false
}

func castXMLMap(inMap map[string]string, schema *openapi3.Schema) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
	for k, v := range inMap {
		ps, ok := getPropertyFromSchema(schema, k)
		if !ok {
			return nil, fmt.Errorf("property missing from schema: '%s'", k)
		}
		castVal, err := castXMLValue(v, ps)
		if err != nil {
			return nil, err
		}
		rv[k] = castVal
	}
	return rv, nil
}

func GetSubObjTyped(xmlReader io.ReadCloser, path string, schema *openapi3.Schema) (interface{}, *xmlquery.Node, error) {
	raw, doc, err := getSubObj(xmlReader, path)
	if err != nil {
		return nil, nil, err
	}
	switch schema.Type {
	case "array":
		if schema.Items == nil || schema.Items.Value == nil {
			return nil, nil, fmt.Errorf("xml serde: cannot accomodate nil items array schema when deserializing an xml array")
		}
		switch raw := raw.(type) {
		case []map[string]string:
			var rv []map[string]interface{}
			for _, m := range raw {
				mc, err := castXMLMap(m, schema.Items.Value)
				if err != nil {
					return nil, nil, err
				}
				rv = append(rv, mc)
			}
			return rv, doc, nil
		default:
			return nil, nil, fmt.Errorf("xml serde: openapi schema type 'array' cannot accomodate golang type '%T'", raw)
		}
	case "object":
		switch raw := raw.(type) {
		case map[string]string:
			mc, err := castXMLMap(raw, schema)
			if err != nil {
				return nil, nil, err
			}
			return []map[string]interface{}{mc}, doc, nil
		default:
			return nil, nil, fmt.Errorf("xml serde: openapi schema type 'object' cannot accomodate golang type '%T'", raw)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported openapi schema type '%s'", schema.Type)
	}
}

func getSubObj(xmlReader io.ReadCloser, path string) (interface{}, *xmlquery.Node, error) {
	doc, err := xmlquery.Parse(xmlReader)
	if err != nil {
		return nil, nil, err
	}
	nodes, err := xmlquery.QueryAll(doc, path)
	if err != nil {
		return nil, nil, err
	}
	if len(nodes) == 1 {
		m, err := getNodeMap(nodes[0])
		if err != nil {
			return nil, nil, err
		}
		rv := []interface{}{m}
		return rv, doc, nil
	}
	var rv []map[string]string
	for _, node := range nodes {
		switch node.Type {
		case xmlquery.TextNode, xmlquery.CharDataNode, xmlquery.CommentNode:
			return node.InnerText(), doc, nil
		default:
			nm, err := getNodeMap(node)
			if err != nil {
				return nil, nil, err
			}
			rv = append(rv, nm)
		}
	}
	return rv, doc, nil
}

func MarshalXMLUserInput(input interface{}, enclosingName string) ([]byte, error) {
	switch input := input.(type) {
	case map[string]interface{}:
		m := newPermissableMapWrapper(input, enclosingName)
		return xml.Marshal(m)
	default:
		return nil, fmt.Errorf("cannot MarshaL XML user input from type = '%T'", input)
	}
}

type permissableMap map[string]interface{}

type permissableMapWrapper struct {
	m    permissableMap
	name xml.Name
}

func newPermissableMapWrapper(m map[string]interface{}, name string) permissableMapWrapper {
	return permissableMapWrapper{
		m:    m,
		name: xml.Name{"", name},
	}
}

func (s permissableMapWrapper) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	start.Name = s.name
	tokens := []xml.Token{start}

	for key, value := range s.m {
		t := xml.StartElement{Name: xml.Name{"", key}}
		tokens = append(tokens, t, xml.CharData(fmt.Sprintf("%v", value)), xml.EndElement{t.Name})
	}

	tokens = append(tokens, xml.EndElement{start.Name})

	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	// flush to ensure tokens are written
	return e.Flush()
}
