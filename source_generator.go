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
	// for k, v := range svc.Components.Schemas {
	// 	sv := cs.Sdump(v.Value)
	// 	f.Write([]byte(fmt.Sprintf("Schema_%s := %s\n\n", k, sv)))
	// }
	// for k, v := range svc.Components.Schemas {
	// 	sv := cs.Sdump(v)
	// 	f.Write([]byte(fmt.Sprintf("Schema_%s := %s\n\n", k, sv)))
	// }
	cs.Fdump(f, svc)
	return nil
}
