package openapistackql

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stackql/stackql-parser/go/sqltypes"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type ITable interface {
	GetName() string
	KeyExists(string) bool
	GetKey(string) (interface{}, error)
	GetKeyAsSqlVal(string) (sqltypes.Value, error)
	GetRequiredParameters() map[string]Addressable
	FilterBy(func(interface{}) (ITable, error)) (ITable, error)
}

type ColumnDescriptor interface {
	GetIdentifier() string
	GetRepresentativeSchema() Schema
	GetName() string
	getSchema() Schema
	setName(string)
}

type standardColumnDescriptor struct {
	Alias        string
	Name         string
	Qualifier    string
	Schema       Schema
	DecoratedCol string
	Val          *sqlparser.SQLVal
	Node         sqlparser.SQLNode
}

func (cd standardColumnDescriptor) setName(name string) {
	cd.Name = name
}

func (cd standardColumnDescriptor) GetName() string {
	return cd.Name
}

func (cd standardColumnDescriptor) getSchema() Schema {
	return cd.Schema
}

func (cd standardColumnDescriptor) GetIdentifier() string {
	if cd.Alias != "" {
		return cd.Alias
	}
	return cd.Name
}

func (cd standardColumnDescriptor) GetRepresentativeSchema() Schema {
	if cd.Node != nil {
		switch nt := cd.Node.(type) {
		case *sqlparser.ConvertExpr:
			if nt.Type != nil && nt.Type.Type != "" {
				return NewSchema(&openapi3.Schema{Type: nt.Type.Type}, cd.Schema.getService(), cd.Schema.getKey(), "")
			}
		// TODO: make this intelligent
		case *sqlparser.FuncExpr:
			return NewSchema(&openapi3.Schema{Type: "string"}, cd.Schema.getService(), cd.Schema.getKey(), "")
		}

	}
	return cd.Schema
}

func NewColumnDescriptor(alias string, name string, qualifier string, decoratedCol string, node sqlparser.SQLNode, schema Schema, val *sqlparser.SQLVal) ColumnDescriptor {
	return newColumnDescriptor(alias, name, qualifier, decoratedCol, node, schema, val)
}

func newColumnDescriptor(alias string, name string, qualifier string, decoratedCol string, node sqlparser.SQLNode, schema Schema, val *sqlparser.SQLVal) ColumnDescriptor {
	return standardColumnDescriptor{Alias: alias, Name: name, Qualifier: qualifier, DecoratedCol: decoratedCol, Schema: schema, Val: val, Node: node}
}

type Tabulation interface {
	GetColumns() []ColumnDescriptor
	GetSchema() Schema
	PushBackColumn(col ColumnDescriptor)
	GetName() string
	RenameColumnsToXml() Tabulation
}

type standardTabulation struct {
	columns   []ColumnDescriptor
	name      string
	arrayType string
	schema    *standardSchema
}

func GetTabulation(name, arrayType string) Tabulation {
	return &standardTabulation{name: name, arrayType: arrayType}
}

func newStandardTabulation(name string, columns []ColumnDescriptor, schema *standardSchema) Tabulation {
	return &standardTabulation{name: name, columns: columns, schema: schema}
}

func (t *standardTabulation) GetColumns() []ColumnDescriptor {
	return t.columns
}

func (t *standardTabulation) GetSchema() Schema {
	return t.schema
}

func (t *standardTabulation) PushBackColumn(col ColumnDescriptor) {
	t.columns = append(t.columns, col)
}

func (t *standardTabulation) GetName() string {
	return t.name
}

func (t *standardTabulation) RenameColumnsToXml() Tabulation {
	for i, v := range t.columns {
		if v.getSchema() != nil {
			alias := v.getSchema().getXmlAlias()
			if alias != "" {
				t.columns[i].setName(alias)
			}
		}
	}
	return t
}
