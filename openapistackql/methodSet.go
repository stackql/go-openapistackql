package openapistackql

type MethodSet []OperationStore

func (ms MethodSet) GetFirstMatch(params map[string]interface{}) (OperationStore, map[string]interface{}, bool) {
	return ms.getFirstMatch(params)
}

func (ms MethodSet) GetFirst() (OperationStore, string, bool) {
	return ms.getFirst()
}

func (ms MethodSet) getFirstMatch(params map[string]interface{}) (OperationStore, map[string]interface{}, bool) {
	for _, m := range ms {
		if remainingParams, ok := m.ParameterMatch(params); ok {
			return m, remainingParams, true
		}
	}
	return nil, params, false
}

func (ms MethodSet) getFirst() (OperationStore, string, bool) {
	for _, m := range ms {
		return m, m.getName(), true
	}
	return nil, "", false
}
