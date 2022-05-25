package xmlmap

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	mxj "github.com/clbanning/mxj/v2"
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/antchfx/xmlquery"
)

/*
This package is all about dealing with the difficulty in XML deserialization.

The problem is stated in [the golang xml package](https://pkg.go.dev/encoding/xml#pkg-note-BUG) as:

> Mapping between XML elements and data structures is inherently flawed: an XML element is an order-dependent collection of anonymous values, while a data structure is an order-independent collection of named values. See package json for a textual representation more suitable to data structures.

*/

var _ mxj.Map

func Unmarshal(xmlReader io.ReadCloser) (mxj.Map, error) {
	return unmarshal(xmlReader)
}

func unmarshal(xmlReader io.ReadCloser) (mxj.Map, error) {
	mv, err := mxj.NewMapXmlReader(xmlReader)
	if err != nil {
		return nil, err
	}
	return mv, nil
}

// for child := n.FirstChild; child != nil; child = child.NextSibling {
// 	output(buf, child)
// }

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

func GetSubObj(xmlReader io.ReadCloser, path string) (interface{}, error) {
	return getSubObj(xmlReader, path)
}

func GetSubObjTyped(xmlReader io.ReadCloser, path string, schema *openapi3.Schema) (interface{}, error) {
	raw, err := getSubObj(xmlReader, path)
	if err != nil {
		return nil, err
	}
	switch schema.Type {
	case "array":
		if schema.Items == nil || schema.Items.Value == nil {
			return nil, fmt.Errorf("xml serde: cannot accomodate nil items array schema when deserializing an xml array")
		}
		switch raw := raw.(type) {
		case []map[string]string:
			var rv []map[string]interface{}
			for _, m := range raw {
				mc, err := castXMLMap(m, schema.Items.Value)
				if err != nil {
					return nil, err
				}
				rv = append(rv, mc)
			}
			return rv, nil
		default:
			return nil, fmt.Errorf("xml serde: openapi schema type 'array' cannot accomodate golang type '%T'", raw)
		}
	case "object":
		switch raw := raw.(type) {
		case map[string]string:
			mc, err := castXMLMap(raw, schema)
			if err != nil {
				return nil, err
			}
			return []map[string]interface{}{mc}, nil
		default:
			return nil, fmt.Errorf("xml serde: openapi schema type 'object' cannot accomodate golang type '%T'", raw)
		}
	default:
		return nil, fmt.Errorf("unsupported openapi schema type '%s'", schema.Type)
	}
}

func GetSubObjArr(xmlReader io.ReadCloser, path string) ([]map[string]interface{}, error) {
	return getSubObjArr(xmlReader, path)
}

func strMapToInterfaceMap(m map[string]string) map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range m {
		rv[k] = v
	}
	return rv
}

func getSubObjArr(xmlReader io.ReadCloser, path string) ([]map[string]interface{}, error) {
	raw, err := getSubObj(xmlReader, path)
	if err != nil {
		return nil, err
	}
	switch raw := raw.(type) {
	case []map[string]string:
		var rv []map[string]interface{}
		for _, v := range raw {
			m := strMapToInterfaceMap(v)
			rv = append(rv, m)
		}
		return rv, nil
	case map[string]string:
		return []map[string]interface{}{strMapToInterfaceMap(raw)}, nil
	default:
		return nil, fmt.Errorf("")
	}
}

func getSubObj(xmlReader io.ReadCloser, path string) (interface{}, error) {
	doc, err := xmlquery.Parse(xmlReader)
	if err != nil {
		return nil, err
	}
	nodes, err := xmlquery.QueryAll(doc, path)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 1 {
		m, err := getNodeMap(nodes[0])
		if err != nil {
			return nil, err
		}
		rv := []interface{}{m}
		return rv, nil
	}
	var rv []map[string]string
	for _, node := range nodes {
		switch node.Type {
		case xmlquery.TextNode, xmlquery.CharDataNode, xmlquery.CommentNode:
			return node.InnerText(), nil
		default:
			nm, err := getNodeMap(node)
			if err != nil {
				return nil, err
			}
			rv = append(rv, nm)
		}
	}
	return rv, nil
}
