module github.com/stackql/go-openapistackql

go 1.17

require (
	github.com/getkin/kin-openapi v0.88.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/jsonpointer v0.19.5
	github.com/magiconair/properties v1.8.5
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stackql/go-spew v1.1.3-alpha24
	github.com/stackql/stackql-provider-registry v0.0.1-alpha8
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	vitess.io/vitess v0.0.9-alpha5
)

require (
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211210111614-af8b64212486 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.43.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
)

replace vitess.io/vitess => github.com/infraql/vitess v0.0.9-alpha5
