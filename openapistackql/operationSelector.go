package openapistackql

var (
	_ OperationSelector = standardOperationSelector{}
)

type OperationSelector interface {
	GetSQLVerb() string
	GetParameters() map[string]interface{}
}

type standardOperationSelector struct {
	SQLVerb string `json:"sqlVerb" yaml:"sqlVerb"` // Required
	// Optional parameters.
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

func (os standardOperationSelector) GetSQLVerb() string {
	return os.SQLVerb
}

func (os standardOperationSelector) GetParameters() map[string]interface{} {
	return os.Parameters
}

func NewOperationSelector(slqVerb string, params map[string]interface{}) OperationSelector {
	return standardOperationSelector{
		SQLVerb:    slqVerb,
		Parameters: params,
	}
}
