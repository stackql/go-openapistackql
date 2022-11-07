package openapistackql

import (
	"github.com/getkin/kin-openapi/openapi3"
	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/sqlparser"
)

type ITable interface {
	GetName() string
	KeyExists(string) bool
	GetKey(string) (interface{}, error)
	GetKeyAsSqlVal(string) (sqltypes.Value, error)
	GetRequiredParameters() map[string]Addressable
	FilterBy(func(interface{}) (ITable, error)) (ITable, error)
}

type ColumnDescriptor struct {
	Alias        string
	Name         string
	Qualifier    string
	Schema       *Schema
	DecoratedCol string
	Val          *sqlparser.SQLVal
	Node         sqlparser.SQLNode
}

func (cd ColumnDescriptor) GetIdentifier() string {
	if cd.Alias != "" {
		return cd.Alias
	}
	return cd.Name
}

func (cd ColumnDescriptor) GetRepresentativeSchema() *Schema {
	if cd.Node != nil {
		switch nt := cd.Node.(type) {
		case *sqlparser.ConvertExpr:
			if nt.Type != nil && nt.Type.Type != "" {
				return NewSchema(&openapi3.Schema{Type: nt.Type.Type}, cd.Schema.svc, cd.Schema.key)
			}
		// TODO: make this intelligent
		case *sqlparser.FuncExpr:
			return NewSchema(&openapi3.Schema{Type: "string"}, cd.Schema.svc, cd.Schema.key)
		}

	}
	return cd.Schema
}

func NewColumnDescriptor(alias string, name string, qualifier string, decoratedCol string, node sqlparser.SQLNode, schema *Schema, val *sqlparser.SQLVal) ColumnDescriptor {
	return newColumnDescriptor(alias, name, qualifier, decoratedCol, node, schema, val)
}

func newColumnDescriptor(alias string, name string, qualifier string, decoratedCol string, node sqlparser.SQLNode, schema *Schema, val *sqlparser.SQLVal) ColumnDescriptor {
	return ColumnDescriptor{Alias: alias, Name: name, Qualifier: qualifier, DecoratedCol: decoratedCol, Schema: schema, Val: val, Node: node}
}

type Tabulation struct {
	columns   []ColumnDescriptor
	name      string
	arrayType string
	schema    *Schema
}

func GetTabulation(name, arrayType string) Tabulation {
	return Tabulation{name: name, arrayType: arrayType}
}

func (t *Tabulation) GetColumns() []ColumnDescriptor {
	return t.columns
}

func (t *Tabulation) GetSchema() *Schema {
	return t.schema
}

func (t *Tabulation) PushBackColumn(col ColumnDescriptor) {
	t.columns = append(t.columns, col)
}

func (t *Tabulation) GetName() string {
	return t.name
}

func (t *Tabulation) RenameColumnsToXml() *Tabulation {
	for i, v := range t.columns {
		if v.Schema != nil {
			alias := v.Schema.getXmlAlias()
			if alias != "" {
				t.columns[i].Name = alias
			}
		}
	}
	return t
}
