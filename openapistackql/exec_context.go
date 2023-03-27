package openapistackql

import (
	"github.com/stackql/go-openapistackql/pkg/internaldto"
)

var (
	_ ExecContext = &standardExecContext{}
)

type ExecContext interface {
	GetExecPayload() internaldto.ExecPayload
	GetResource() Resource
}

type standardExecContext struct {
	execPayload internaldto.ExecPayload
	resource    Resource
}

func (ec *standardExecContext) GetExecPayload() internaldto.ExecPayload {
	return ec.execPayload
}

func (ec *standardExecContext) GetResource() Resource {
	return ec.resource
}

func NewExecContext(payload internaldto.ExecPayload, rsc Resource) ExecContext {
	return &standardExecContext{
		execPayload: payload,
		resource:    rsc,
	}
}
