package util

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/stackql/go-openapistackql/pkg/response"
)

func InterfaceToBytes(subject interface{}, isErrorCol bool) []byte {
	switch sub := subject.(type) {
	case bool:
		if sub == true {
			return []byte("true")
		}
		return []byte("false")
	case string:
		return []byte(sub)
	case int:
		return []byte(strconv.Itoa(sub))
	case int64:
		return []byte(strconv.FormatInt(sub, 10))
	case float32:
		return []byte(fmt.Sprintf("%f", sub))
	case float64:
		return []byte(fmt.Sprintf("%f", sub))
	case []interface{}:
		str, err := json.Marshal(subject)
		if err == nil {
			return []byte(str)
		}
		return []byte(fmt.Sprintf(`{ "marshallingError": {"type": "array", "error": "%s"}}`, err.Error()))
	case map[string]interface{}:
		str, err := json.Marshal(subject)
		if err == nil {
			return []byte(str)
		}
		return []byte(fmt.Sprintf(`{ "marshallingError": {"type": "array", "error": "%s"}}`, err.Error()))
	case nil:
		return []byte("null")
	case response.Response:
		if isErrorCol {
			return []byte(sub.Error())
		}
		return []byte(sub.String())
	default:
		return []byte(fmt.Sprintf(`{ "displayError": {"type": "%T", "error": "currently unable to represent object of type %T"}}`, subject, subject))
	}
}

func IsNil(arg interface{}) bool {
	return arg == nil || reflect.ValueOf(arg).IsNil()
}
