package openapistackql

import (
	"os"

	"github.com/stackql/go-spew/spew"
)

func getSpewConfig() *spew.ConfigState {
	cs := spew.NewDefaultConfig()
	cs.DisableCapacities = true
	cs.DisablePointerAddresses = true
	cs.AsGolangSource = true
	return cs
}

func (svc *Service) AsSourceString() string {
	cs := getSpewConfig()
	return cs.Sdump(svc)
}

func (svc *Service) ToSourceFile(outFile string) error {
	f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	cs := getSpewConfig()
	cs.Fdump(f, svc)
	return nil
}
