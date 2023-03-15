package openapistackql

import (
	"fmt"
)

type Methods map[string]standardOperationStore

func (ms Methods) FindMethod(key string) (OperationStore, error) {
	if m, ok := ms[key]; ok {
		return &m, nil
	}
	return nil, fmt.Errorf("could not find method for key = '%s'", key)
}

func (ms Methods) OrderMethods() ([]OperationStore, error) {
	var selectBin, insertBin, deleteBin, updateBin, execBin []OperationStore
	for k, pv := range ms {
		v := pv
		switch v.GetSQLVerb() {
		case "select":
			v.setMethodKey(k)
			selectBin = append(selectBin, &v)
		case "insert":
			v.setMethodKey(k)
			insertBin = append(insertBin, &v)
		case "update":
			v.setMethodKey(k)
			updateBin = append(updateBin, &v)
		case "delete":
			v.setMethodKey(k)
			deleteBin = append(deleteBin, &v)
		case "exec":
			v.setMethodKey(k)
			execBin = append(execBin, &v)
		default:
			v.setMethodKey(k)
			v.setSQLVerb("exec")
			execBin = append(execBin, &v)
		}
	}
	sortOperationStoreSlices(selectBin, insertBin, deleteBin, updateBin, execBin)
	rv := combineOperationStoreSlices(selectBin, insertBin, deleteBin, updateBin, execBin)
	return rv, nil
}

func (ms Methods) FindFromSelector(sel OperationSelector) (OperationStore, error) {
	for _, m := range ms {
		if m.GetSQLVerb() == sel.GetSQLVerb() {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("could not locate operation for sql verb  = %s", sel.GetSQLVerb())
}
