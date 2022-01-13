package openapistackql

import (
	"fmt"
)

func GetDescribeHeader(extended bool) []string {
	var retVal []string
	if extended {
		retVal = []string{
			"name",
			"type",
			"description",
		}
	} else {
		retVal = []string{
			"name",
			"type",
		}
	}
	return retVal
}

func GetServicesHeader(extended bool) []string {
	var retVal []string
	if extended {
		retVal = []string{
			"id",
			"name",
			"title",
			"description",
			"version",
			"preferred",
		}
	} else {
		retVal = []string{
			"id",
			"name",
			"title",
		}
	}
	return retVal
}

func GetResourcesHeader(extended bool) []string {
	var retVal []string
	if extended {
		retVal = []string{
			"name",
			"id",
			"type",
			"description",
		}
	} else {
		retVal = []string{
			"name",
			"id",
			"type",
		}
	}
	return retVal
}

type MetadataStore struct {
	Store map[string]*Service
}

func (ms *MetadataStore) GetServices() ([]*Service, error) {
	var retVal []*Service
	for _, svc := range ms.Store {
		retVal = append(retVal, svc)
	}
	return retVal, nil
}

func (ms *MetadataStore) GetResources(serviceName string) (map[string]*Resource, error) {
	svc, ok := ms.Store[serviceName]
	if !ok {
		return nil, fmt.Errorf("cannnot find service %s", serviceName)
	}
	return svc.GetResources()
}

func (ms *MetadataStore) GetResource(serviceName string, resourceName string) (*Resource, error) {
	rscs, err := ms.GetResources(serviceName)
	if err != nil {
		return nil, err
	}
	rsc, ok := rscs[resourceName]
	if !ok {
		return nil, fmt.Errorf("cannnot find resource %s", resourceName)
	}
	return rsc, nil
}

type AuthMetadata struct {
	Principal string `json:"principal"`
	Type      string `json:"type"`
	Source    string `json:"source"`
}

func (am *AuthMetadata) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"principal": am.Principal,
		"type":      am.Type,
		"source":    am.Source,
	}
}

func (am *AuthMetadata) GetHeaders() []string {
	return []string{
		"principal",
		"type",
		"source",
	}
}
