package openapistackql

import (
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

var (
	_ jsonpointer.JSONPointable = (AuthDTO)(standardAuthDTO{})
	_ AuthDTO                   = standardAuthDTO{}
)

type AuthDTO interface {
	JSONLookup(token string) (interface{}, error)
	GetType() string
	GetKeyID() string
	GetKeyIDEnvVar() string
	GetKeyFilePath() string
	GetKeyEnvVar() string
	GetScopes() []string
	GetValuePrefix() string
}

type standardAuthDTO struct {
	Scopes      []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Type        string   `json:"type" yaml:"type"`
	ValuePrefix string   `json:"valuePrefix" yaml:"valuePrefix"`
	KeyID       string   `json:"keyID" yaml:"keyID"`
	KeyIDEnvVar string   `json:"keyIDenvvar" yaml:"keyIDenvvar"`
	KeyFilePath string   `json:"credentialsfilepath" yaml:"credentialsfilepath"`
	KeyEnvVar   string   `json:"credentialsenvvar" yaml:"credentialsenvvar"`
}

func (qt standardAuthDTO) GetType() string {
	return qt.Type
}

func (qt standardAuthDTO) GetKeyID() string {
	return qt.KeyID
}

func (qt standardAuthDTO) GetKeyIDEnvVar() string {
	return qt.KeyIDEnvVar
}

func (qt standardAuthDTO) GetKeyFilePath() string {
	return qt.KeyFilePath
}

func (qt standardAuthDTO) GetKeyEnvVar() string {
	return qt.KeyEnvVar
}

func (qt standardAuthDTO) GetScopes() []string {
	return qt.Scopes
}

func (qt standardAuthDTO) GetValuePrefix() string {
	return qt.ValuePrefix
}

func (qt standardAuthDTO) JSONLookup(token string) (interface{}, error) {
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
