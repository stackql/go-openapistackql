package openapistackql

import (
	"github.com/stackql/go-suffix-map/pkg/suffixmap"
)

type ParameterSuffixMap struct {
	sm suffixmap.SuffixMap
}

func NewParameterSuffixMap() *ParameterSuffixMap {
	return &ParameterSuffixMap{
		sm: suffixmap.NewSuffixMap(nil),
	}
}

func MakeSuffixMapFromParameterMap(m map[string]*Parameter) *ParameterSuffixMap {
	m2 := make(map[string]interface{})
	for k, v := range m {
		m2[k] = v
	}
	return &ParameterSuffixMap{
		sm: suffixmap.NewSuffixMap(m2),
	}
}

func (psm *ParameterSuffixMap) Get(k string) (*Parameter, bool) {
	rv, ok := psm.sm.Get(k)
	if !ok {
		return nil, false
	}
	crv, ok := rv.(*Parameter)
	return crv, ok
}

func (psm *ParameterSuffixMap) GetAll() map[string]*Parameter {
	m := psm.sm.GetAll()
	rv := make(map[string]*Parameter)
	for k, v := range m {
		p, ok := v.(*Parameter)
		if ok {
			rv[k] = p
		}
	}
	return rv
}

func (psm *ParameterSuffixMap) Put(k string, v Addressable) {
	psm.sm.Put(k, v)
}

func (psm *ParameterSuffixMap) Delete(k string) bool {
	return psm.sm.Delete(k)
}

func (psm *ParameterSuffixMap) Size() int {
	return psm.sm.Size()
}
