package querytranspose

import (
	"encoding/json"
	"fmt"
)

type AWSCanonicalQueryTransposer struct {
	baseKey  string
	rawInput []byte
}

func newAWSCanonicalQueryTransposer(rawInput []byte, baseKey string) QueryTransposer {
	return &AWSCanonicalQueryTransposer{
		rawInput: rawInput,
		baseKey:  baseKey,
	}
}

func (um *AWSCanonicalQueryTransposer) Transpose() (map[string]string, error) {
	var v interface{}
	err := json.Unmarshal(um.rawInput, &v)
	if err != nil {
		return nil, err
	}
	switch input := v.(type) {
	case []interface{}:
		return unmarshalSlice(um.baseKey, input)
	case map[string]interface{}:
		return unmarshalMap(um.baseKey, input)
	case int, int32, int64, float32, float64, string:
		return map[string]string{
			um.baseKey: fmt.Sprintf("%v", input),
		}, nil
	default:
		return nil, fmt.Errorf("cannot unmarshal AWS Canonical query param of type = '%T'", input)
	}
}

func unmarshalMap(baseKey string, m map[string]interface{}) (map[string]string, error) {
	rv := make(map[string]string)
	for k, v := range m {
		kJoined := fmt.Sprintf("%s.%s", baseKey, k)
		switch v := v.(type) {
		case []interface{}:
			iv, err := unmarshalSlice(kJoined, v)
			if err != nil {
				return nil, err
			}
			for subK, subV := range iv {
				rv[subK] = subV
			}
		case map[string]interface{}:
			sv, err := unmarshalMap(kJoined, v)
			if err != nil {
				return nil, err
			}
			for subK, subV := range sv {
				rv[subK] = subV
			}
		default:
			rv[kJoined] = fmt.Sprintf("%v", v)
		}
	}
	return rv, nil
}

func unmarshalSlice(baseKey string, s []interface{}) (map[string]string, error) {
	rv := make(map[string]string)
	for i, v := range s {
		kJoined := fmt.Sprintf("%s.%d", baseKey, i+1)
		switch v := v.(type) {
		case []interface{}:
			iv, err := unmarshalSlice(kJoined, v)
			if err != nil {
				return nil, err
			}
			for subK, subV := range iv {
				rv[subK] = subV
			}
		case map[string]interface{}:
			sv, err := unmarshalMap(kJoined, v)
			if err != nil {
				return nil, err
			}
			for subK, subV := range sv {
				rv[subK] = subV
			}
		default:
			rv[kJoined] = fmt.Sprintf("%v", v)
		}
	}
	return rv, nil
}
