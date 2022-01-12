module github.com/stackql/openapistackql

go 1.16

require (
	github.com/stackql/go-spew v1.1.3-alpha15
	github.com/getkin/kin-openapi v0.88.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/jsonpointer v0.19.5
	github.com/magiconair/properties v1.8.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.5.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	vitess.io/vitess v0.0.9-alpha5
)

replace vitess.io/vitess => github.com/infraql/vitess v0.0.9-alpha5
