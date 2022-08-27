package requesttranslate

import (
	"net/http"
)

func NewNilTranslator() RequestTranslator {
	return &NilTranslator{}
}

type NilTranslator struct {
}

func (gp *NilTranslator) Translate(req *http.Request) (*http.Request, error) {
	return req, nil
}
