module github.com/stackql/go-openapistackql

go 1.17

require (
	github.com/getkin/kin-openapi v0.88.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/jsonpointer v0.19.5
	github.com/magiconair/properties v1.8.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/stackql/go-spew v1.1.3-alpha24
	github.com/stretchr/testify v1.5.1
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	vitess.io/vitess v0.0.9-alpha5
)

require (
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mitchellh/mapstructure v1.2.3 // indirect
	github.com/pelletier/go-toml v1.2.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.0.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20190926190326-7ee9db18f195 // indirect
	google.golang.org/grpc v1.24.0 // indirect
)

replace vitess.io/vitess => github.com/infraql/vitess v0.0.9-alpha5
