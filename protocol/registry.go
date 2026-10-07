package protocol

import "github.com/tapirtests/core/model"

// Registry holds the drivers available to a run, by protocol. The shell
// (CLI, local server) builds it and passes it to the engine; that is how
// the engine stays independent of concrete protocols.
//
// A Registry is filled before a run and only read during it.
type Registry struct {
	drivers map[model.Protocol]Driver
}

// NewRegistry returns a registry with the given drivers.
func NewRegistry(drivers ...Driver) *Registry {
	r := &Registry{drivers: make(map[model.Protocol]Driver, len(drivers))}
	for _, d := range drivers {
		r.Register(d)
	}
	return r
}

// Register adds a driver. A driver registered earlier for the same protocol
// is replaced: tests use this to swap a real driver for a scripted one.
func (r *Registry) Register(d Driver) {
	r.drivers[d.Protocol()] = d
}

// Driver returns the driver of a protocol.
func (r *Registry) Driver(p model.Protocol) (Driver, bool) {
	if r == nil {
		return nil, false
	}
	d, ok := r.drivers[p]
	return d, ok
}
