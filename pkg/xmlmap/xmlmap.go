package xmlmap

import (
	"fmt"
	"io"
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

func getNodeMap(node *xmlquery.Node) (map[string]interface{}, error) {
	rv := make(map[string]interface{})
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

func GetSubObj(xmlReader io.ReadCloser, path string) (interface{}, error) {
	return getSubObj(xmlReader, path)
}

func GetSubObjTyped(xmlReader io.ReadCloser, path string, schema *openapi3.Schema) (interface{}, error) {
	rv, err := getSubObj(xmlReader, path)
	if err != nil {
		return rv, err
	}
	switch schema.Type {
	case "array":
		return nil, fmt.Errorf("unsupported openapi schema type '%s'", schema.Type)
	case "object":
		return nil, fmt.Errorf("unsupported openapi schema type '%s'", schema.Type)
	default:
		return nil, fmt.Errorf("unsupported openapi schema type '%s'", schema.Type)
	}
}

func GetSubObjArr(xmlReader io.ReadCloser, path string) ([]map[string]interface{}, error) {
	return getSubObjArr(xmlReader, path)
}

func getSubObjArr(xmlReader io.ReadCloser, path string) ([]map[string]interface{}, error) {
	rv, err := getSubObj(xmlReader, path)
	if err != nil {
		return nil, err
	}
	switch rv := rv.(type) {
	case []map[string]interface{}:
		return rv, nil
	case map[string]interface{}:
		return []map[string]interface{}{rv}, nil
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
	var rv []map[string]interface{}
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
