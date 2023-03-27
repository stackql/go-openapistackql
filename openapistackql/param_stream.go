package openapistackql

import (
	"io"
)

type ParamInputStrem interface {
	Read() ([]HttpParameters, error)
}

type standardParamInputStrem struct {
	store []HttpParameters
}

func NewStandardParamInputStrem() ParamInputStrem {
	return &standardParamInputStrem{}
}

func (ss *standardParamInputStrem) Read() ([]HttpParameters, error) {
	rv := ss.store
	ss.store = nil
	return rv, io.EOF
}
