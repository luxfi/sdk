// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

// VMFactory creates VM instances
type VMFactory interface {
	// New creates a new VM instance
	New() (interface{}, error)
}

// PluginRegistry manages VM plugins
type PluginRegistry interface {
	// Register registers a VM factory with a name
	Register(name string, factory VMFactory) error
	
	// Get retrieves a VM factory by name
	Get(name string) (VMFactory, error)
	
	// List returns all registered VM names
	List() []string
}

// GlobalRegistry is the global plugin registry
var GlobalRegistry PluginRegistry = &defaultRegistry{
	factories: make(map[string]VMFactory),
}

type defaultRegistry struct {
	factories map[string]VMFactory
}

func (r *defaultRegistry) Register(name string, factory VMFactory) error {
	r.factories[name] = factory
	return nil
}

func (r *defaultRegistry) Get(name string) (VMFactory, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, nil
	}
	return factory, nil
}

func (r *defaultRegistry) List() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// RegisterPlugin registers a VM plugin
func RegisterPlugin(name string, factory VMFactory) error {
	return GlobalRegistry.Register(name, factory)
}