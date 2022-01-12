package openapistackql

import (
	"os"

	"github.com/davecgh/go-spew/spew"
)

func (svc *Service) AsSourceString() string {
	return spew.Sdump(svc)
}

func (svc *Service) ToSourceFile(outFile string) error {
	f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	spew.Fdump(f, svc)
	return nil
}
