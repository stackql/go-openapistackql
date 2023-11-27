package openapistackql

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/stackql/go-openapistackql/pkg/constants"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type requestBodyParam struct {
	Key string
	Val interface{}
}

func parseRequestBodyParam(k string, v interface{}, s Schema) *requestBodyParam {
	trimmedKey := strings.TrimPrefix(k, constants.RequestBodyBaseKey)
	var parsedVal interface{}
	if trimmedKey != k { //nolint:nestif // keep for now
		switch vt := v.(type) {
		case string:
			var isStringRestricted bool
			if s != nil {
				isStringRestrictedRaw, hasStr := s.getExtension(ExtensionKeyStringOnly)
				if hasStr {
					boolStr, isBoolStr := isStringRestrictedRaw.(string)
					if isBoolStr && boolStr == "true" {
						isStringRestricted = true
					}
				}
			}
			var js map[string]interface{}
			var jArr []interface{}
			//nolint:gocritic // keep for now
			if isStringRestricted {
				parsedVal = vt
			} else if json.Unmarshal([]byte(vt), &js) == nil {
				parsedVal = js
			} else if json.Unmarshal([]byte(vt), &jArr) == nil {
				parsedVal = jArr
			} else {
				parsedVal = vt
			}
		case *sqlparser.FuncExpr:
			if strings.ToLower(vt.Name.GetRawVal()) == "string" && len(vt.Exprs) == 1 {
				pv, err := getStringFromStringFunc(vt)
				if err == nil {
					parsedVal = pv
				} else {
					parsedVal = vt
				}
			} else {
				parsedVal = vt
			}
		default:
			parsedVal = vt
		}
		return &requestBodyParam{
			Key: trimmedKey,
			Val: parsedVal,
		}
	}
	return nil
}

//nolint:gocognit // not super complex
func splitHTTPParameters(
	sqlParamMap map[int]map[string]interface{},
	method OperationStore,
) ([]HttpParameters, error) {
	var retVal []HttpParameters
	var rowKeys []int
	requestSchema, _ := method.GetRequestBodySchema()
	responseSchema, _ := method.GetRequestBodySchema()
	for idx := range sqlParamMap {
		rowKeys = append(rowKeys, idx)
	}
	sort.Ints(rowKeys)
	for _, key := range rowKeys {
		sqlRow := sqlParamMap[key]
		reqMap := NewHttpParameters(method)
		for k, v := range sqlRow {
			if param, ok := method.GetOperationParameter(k); ok {
				reqMap.StoreParameter(param, v)
			} else {
				if requestSchema != nil {
					kCleaned := strings.TrimPrefix(k, RequestBodyBaseKey)
					prop, _ := requestSchema.GetProperty(kCleaned)
					rbp := parseRequestBodyParam(k, v, prop)
					if rbp != nil {
						reqMap.SetRequestBodyParam(rbp.Key, rbp.Val)
						continue
					}
				}
				reqMap.SetServerParam(k, method.GetService(), v)
			}
			if responseSchema != nil && responseSchema.FindByPath(k, nil) != nil {
				reqMap.SetResponseBodyParam(k, v)
			}
		}
		retVal = append(retVal, reqMap)
	}
	return retVal, nil
}

func getStringFromStringFunc(fe *sqlparser.FuncExpr) (string, error) {
	if strings.ToLower(fe.Name.GetRawVal()) == "string" && len(fe.Exprs) == 1 {
		//nolint:gocritic // acceptable
		switch et := fe.Exprs[0].(type) {
		case *sqlparser.AliasedExpr:
			switch et2 := et.Expr.(type) {
			case *sqlparser.SQLVal:
				return string(et2.Val), nil
			}
		}
	}
	return "", fmt.Errorf("cannot extract string from func '%s'", fe.Name)
}
