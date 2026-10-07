package protocol_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/protocol"
)

// stubDriver is a driver of the given protocol that returns a fixed status.
type stubDriver struct {
	proto  model.Protocol
	status int
}

func (d stubDriver) Protocol() model.Protocol { return d.proto }

func (d stubDriver) Execute(context.Context, *protocol.Request) (*protocol.Result, error) {
	return &protocol.Result{Status: d.status}, nil
}

func TestRegistry(t *testing.T) {
	r := protocol.NewRegistry(stubDriver{proto: model.ProtocolHTTP, status: 1})

	d, ok := r.Driver(model.ProtocolHTTP)
	if !ok {
		t.Fatal("registered driver is not found")
	}
	if res, _ := d.Execute(context.Background(), &protocol.Request{}); res.Status != 1 {
		t.Errorf("got the driver with status %d, want 1", res.Status)
	}

	if _, ok := r.Driver("graphql"); ok {
		t.Error("a driver is found for an unregistered protocol")
	}
}

func TestRegistryRegisterReplaces(t *testing.T) {
	r := protocol.NewRegistry(stubDriver{proto: model.ProtocolHTTP, status: 1})
	r.Register(stubDriver{proto: model.ProtocolHTTP, status: 2})
	r.Register(stubDriver{proto: "graphql", status: 3})

	for proto, want := range map[model.Protocol]int{model.ProtocolHTTP: 2, "graphql": 3} {
		d, ok := r.Driver(proto)
		if !ok {
			t.Fatalf("no driver for %q", proto)
		}
		if res, _ := d.Execute(context.Background(), &protocol.Request{}); res.Status != want {
			t.Errorf("%q: driver with status %d, want %d", proto, res.Status, want)
		}
	}
}

func TestEmptyAndNilRegistry(t *testing.T) {
	if _, ok := protocol.NewRegistry().Driver(model.ProtocolHTTP); ok {
		t.Error("an empty registry has a driver")
	}
	var r *protocol.Registry
	if _, ok := r.Driver(model.ProtocolHTTP); ok {
		t.Error("a nil registry has a driver")
	}
}

func TestError(t *testing.T) {
	cause := context.DeadlineExceeded
	err := protocol.Errorf(protocol.ErrTimeout, "no response in 30s: %w", cause)

	if got, want := err.Error(), "timeout: no response in 30s: context deadline exceeded"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is does not see the underlying error")
	}

	// A driver error stays recognizable after being wrapped by the caller.
	var wrapped error = errors.Join(errors.New("step failed"), err)
	var perr *protocol.Error
	if !errors.As(wrapped, &perr) || perr.Kind != protocol.ErrTimeout {
		t.Errorf("errors.As: kind = %v", perr)
	}

	if got := (&protocol.Error{Kind: protocol.ErrCanceled}).Error(); got != "canceled" {
		t.Errorf("Error() without a cause = %q", got)
	}
}
