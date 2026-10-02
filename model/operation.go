package model

// Operation is the protocol-specific part of a RequestDef: what exactly to
// call (for HTTP — a method and a path). It is data only; the code that
// performs the call is a protocol driver, chosen by Protocol().
type Operation interface {
	Protocol() Protocol
}

// Protocol identifies the protocol of an Operation and selects the driver
// that executes it, e.g. "http".
type Protocol string

const (
	// ProtocolHTTP is plain HTTP (REST).
	ProtocolHTTP Protocol = "http"
)

// HTTPOperation is an HTTP (REST) operation: a method and a path template.
// Inputs of the request are mapped onto it by name prefix: path.<name> fills
// {name} in Path, query.*, header.*, cookie.* and body.* go to the respective
// parts of the request.
type HTTPOperation struct {
	Method   string // GET, POST, PUT, PATCH, DELETE, ...
	Path     string // path template relative to the base URL, e.g. /products/{id}
	Consumes string // request body content type: application/json, multipart/form-data, ...
}

// Protocol implements Operation.
func (HTTPOperation) Protocol() Protocol { return ProtocolHTTP }

// Compile-time check that HTTPOperation implements Operation.
var _ Operation = HTTPOperation{}
