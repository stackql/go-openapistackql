package openapistackql

import (
	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/sqlparser"
)

type ITable interface {
	GetName() string
	KeyExists(string) bool
	GetKey(string) (interface{}, error)
	GetKeyAsSqlVal(string) (sqltypes.Value, error)
	GetRequiredParameters() map[string]*Parameter
	FilterBy(func(interface{}) (ITable, error)) (ITable, error)
}

type ColumnDescriptor struct {
	Alias        string
	Name         string
	Schema       *Schema
	DecoratedCol string
	Val          *sqlparser.SQLVal
}

func (cd ColumnDescriptor) GetIdentifier() string {
	if cd.Alias != "" {
		return cd.Alias
	}
	return cd.Name
}

func NewColumnDescriptor(alias string, name string, decoratedCol string, schema *Schema, val *sqlparser.SQLVal) ColumnDescriptor {
	return ColumnDescriptor{Alias: alias, Name: name, DecoratedCol: decoratedCol, Schema: schema, Val: val}
}

type Tabulation struct {
	columns   []ColumnDescriptor
	name      string
	arrayType string
}

func GetTabulation(name, arrayType string) Tabulation {
	return Tabulation{name: name, arrayType: arrayType}
}

func (t *Tabulation) GetColumns() []ColumnDescriptor {
	return t.columns
}

func (t *Tabulation) PushBackColumn(col ColumnDescriptor) {
	t.columns = append(t.columns, col)
}

func (t *Tabulation) GetName() string {
	return t.name
}
