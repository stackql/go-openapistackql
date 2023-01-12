package openapistackql_test

import (
	"testing"

	"github.com/stackql/go-openapistackql/openapistackql"
	"gopkg.in/yaml.v3"

	"gotest.tools/assert"
)

var (
	simpleYamlLoadTestInput string = `
predicate: sqlDialect == "sqlite3"
ddl: select * from someprovider.someservice.someresource
fallback:
    ddl: select * from someprovider.someservice.someresource where x = true
`
	noFallbackYamlLoadTestInput string = `
predicate: sqlDialect == "sqlite3"
ddl: select * from someprovider.someservice.someresource
`
)

func TestSimpleViewApi(t *testing.T) {

	var v openapistackql.View
	err := yaml.Unmarshal([]byte(simpleYamlLoadTestInput), &v)
	if err != nil {
		t.Fatalf("TestSimpleViewApi failed at unmarshal step, err = '%s'", err.Error())
	}

	ddlForSQLite3, ok := v.GetDDLForSqlDialect("sqlite3")
	if !ok {
		t.Fatalf("TestSimpleViewApi failed at get DDL for sqlite3 step")
	}
	assert.Assert(t, ddlForSQLite3 == "select * from someprovider.someservice.someresource")

	ddlForPostgres, ok := v.GetDDLForSqlDialect("postgres")
	if !ok {
		t.Fatalf("TestSimpleViewApi failed at get DDL for postgres step")
	}
	assert.Assert(t, ddlForPostgres == "select * from someprovider.someservice.someresource where x = true")

	t.Logf("TestSimpleViewApi passed")
}

func TestNoFallbackViewApi(t *testing.T) {

	var v openapistackql.View
	err := yaml.Unmarshal([]byte(noFallbackYamlLoadTestInput), &v)
	if err != nil {
		t.Fatalf("TestNoFallbackViewApi failed at unmarshal step, err = '%s'", err.Error())
	}

	ddlForSQLite3, ok := v.GetDDLForSqlDialect("sqlite3")
	if !ok {
		t.Fatalf("TestNoFallbackViewApi failed at get DDL for sqlite3 step")
	}
	assert.Assert(t, ddlForSQLite3 == "select * from someprovider.someservice.someresource")

	ddlForPostgres, ok := v.GetDDLForSqlDialect("postgres")
	if ok {
		t.Fatalf("TestNoFallbackViewApi failed at get DDL for postgres step; should **NOT** receive any DDL")
	}
	assert.Assert(t, ddlForPostgres == "")

	t.Logf("TestNoFallbackViewApi passed")
}
