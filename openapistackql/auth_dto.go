package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type AuthDTO struct {
	Scopes      []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Type        string   `json:"type" yaml:"type"`
	ValuePrefix string   `json:"valuePrefix" yaml:"valuePrefix"`
	KeyID       string   `json:"keyID" yaml:"keyID"`
	KeyIDEnvVar string   `json:"keyIDenvvar" yaml:"keyIDenvvar"`
	KeyFilePath string   `json:"credentialsfilepath" yaml:"credentialsfilepath"`
	KeyEnvVar   string   `json:"credentialsenvvar" yaml:"credentialsenvvar"`
}

var _ jsonpointer.JSONPointable = (AuthDTO)(AuthDTO{})

func (qt AuthDTO) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "keyID":
		return qt.KeyID, nil
	case "credentialsfilepath":
		return qt.KeyFilePath, nil
	case "credentialsenvvar":
		return qt.KeyEnvVar, nil
	case "keyIDenvvar":
		return qt.KeyIDEnvVar, nil
	case "valuePrefix":
		return qt.ValuePrefix, nil
	case "type":
		return qt.Type, nil
	case "scopes":
		return qt.Scopes, nil
	default:
		return nil, fmt.Errorf("could not resolve token '%s' from AuthDTO doc object", token)
	}
}
