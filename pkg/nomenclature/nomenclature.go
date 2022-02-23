package nomenclature

import (
	"github.com/stackql/stackql-provider-registry/registry/pkg/nomenclature"
)

type ProviderDesignation nomenclature.ProviderDesignation

func ExtractProviderDesignation(providerStr string) (ProviderDesignation, error) {
	rv, err := nomenclature.ExtractProviderDesignation(providerStr)
	return ProviderDesignation(rv), err
}
