package openapistackql

var (
	_ ResourceRegister = &standardResourceRegister{}
)

type ResourceRegister interface {
	//
	GetServiceDocPath() *ServiceRef
	ObtainServiceDocUrl(resourceKey string) string
	SetProviderService(ps ProviderService)
	SetProvider(p Provider)
	GetResources() map[string]Resource
	GetResource(string) (Resource, bool)
	//
	getProvider() Provider
	getProviderService() ProviderService
	setOpStore(resourceString string, methodString string, opStore OperationStore)
}

func NewResourceRegister() ResourceRegister {
	return newStandardResourceRegister()
}

func newStandardResourceRegister() *standardResourceRegister {
	return &standardResourceRegister{
		ServiceDocPath: &ServiceRef{},
		Resources:      make(map[string]*standardResource),
	}
}

type standardResourceRegister struct {
	ServiceDocPath  *ServiceRef                  `json:"serviceDoc,omitempty" yaml:"serviceDoc,omitempty"`
	Resources       map[string]*standardResource `json:"resources,omitempty" yaml:"resources,omitempty"`
	ProviderService ProviderService              `json:"-" yaml:"-"` // upwards traversal
	Provider        Provider                     `json:"-" yaml:"-"` // upwards traversal
}

func (rr *standardResourceRegister) GetResource(resourceKey string) (Resource, bool) {
	rsc, ok := rr.Resources[resourceKey]
	return rsc, ok
}

func (rr *standardResourceRegister) setOpStore(resourceString string, methodString string, opStore OperationStore) {
	rr.Resources[resourceString].setMethod(methodString, opStore.(*standardOperationStore))
}

func (rr *standardResourceRegister) getProvider() Provider {
	return rr.Provider
}

func (rr *standardResourceRegister) getProviderService() ProviderService {
	return rr.ProviderService
}

func (rr *standardResourceRegister) GetResources() map[string]Resource {
	rv := make(map[string]Resource, len(rr.Resources))
	for k, v := range rr.Resources {
		rv[k] = v
	}
	return rv
}

func (rr *standardResourceRegister) SetProviderService(ps ProviderService) {
	rr.ProviderService = ps
}

func (rr *standardResourceRegister) GetServiceDocPath() *ServiceRef {
	return rr.ServiceDocPath
}

func (rr *standardResourceRegister) SetProvider(p Provider) {
	rr.Provider = p
}

func (rr *standardResourceRegister) ObtainServiceDocUrl(resourceKey string) string {
	var rv string
	if rr.ServiceDocPath != nil {
		rv = rr.ServiceDocPath.Ref
	}
	rsc, ok := rr.Resources[resourceKey]
	if ok && rsc.GetServiceDocPath() != nil && rsc.GetServiceDocPath().Ref != "" {
		rv = rsc.GetServiceDocPath().Ref
	}
	return rv
}
