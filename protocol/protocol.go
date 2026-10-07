// Package protocol defines how the engine talks to an API: a Driver performs
// one call of a request and returns a normalized result.
//
// The engine knows nothing about HTTP or any other protocol. It resolves the
// inputs of a step, picks the driver by the protocol of the request operation
// and hands the call over. Everything protocol-specific — building a URL,
// placing credentials, encoding a body, parsing a response — lives in a
// driver (protocol/http, later others). Tests use a scripted driver instead
// of the network.
//
// This package covers unary calls: one call, one response. Streaming
// protocols need a different kind of step and will get their own interface.
package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/tapirtests/core/model"
)

// Driver performs calls of one protocol.
//
// Execute returns a Result whenever a response was received, whatever its
// status: a 404 or a 500 is a result for assertions to judge, not an error.
// It returns an error only when there is no response to judge; the error
// should be an *Error telling why.
//
// A driver must be safe for concurrent use and must stop when ctx is done:
// the engine sets the timeout of a call on ctx.
type Driver interface {
	// Protocol tells which operations the driver executes.
	Protocol() model.Protocol
	Execute(ctx context.Context, req *Request) (*Result, error)
}

// Request is one call to perform. Everything in it is already resolved:
// templates are evaluated, values are final.
type Request struct {
	// RequestID identifies the request definition; used in messages.
	RequestID model.RequestID
	// Operation is the protocol-specific part of the definition. Its dynamic
	// type matches the driver: an HTTP driver gets a model.HTTPOperation.
	Operation model.Operation
	// BaseURL is the resolved base address of the API.
	BaseURL string
	// Inputs are the values passed by the step, by input name of the request
	// contract: "path.id", "body.name", "auth.bearer". A key with a nil
	// value is an explicit null ("intentionally no value"); a missing key is
	// an optional input that was not passed.
	Inputs map[string]any
	// Schemes are the security schemes of the project by name. The driver
	// uses them to place the values of "auth.<scheme>" inputs.
	Schemes map[string]*model.SecurityScheme
}

// Result is a received response in a protocol-independent form.
type Result struct {
	// Status is the status of the response: the HTTP status code.
	Status int
	// Headers are the response headers. Names are lower-case, because header
	// names are case-insensitive and paths into the response are not;
	// repeated headers are joined with ", ".
	Headers map[string]string
	// Body is the parsed body: for JSON, the value produced by decoding with
	// numbers kept as json.Number; a string for other text; nil if empty.
	Body any
	// Raw is the body as received.
	Raw []byte
	// Duration is how long the call took, from sending the request to
	// reading the whole response.
	Duration time.Duration
	// Sent describes the request as it actually went out, for the report.
	Sent Sent
}

// Sent is what a driver actually sent, in a form suitable for a report.
// Secrets in it are masked later, by the engine.
type Sent struct {
	// Line is a one-line summary of the call: "POST http://api:8080/products".
	Line    string
	Headers map[string]string
	Body    any
}

// ErrorKind tells why a call produced no response.
type ErrorKind string

const (
	// ErrInvalidRequest: the request could not be built from the inputs,
	// e.g. a value cannot be placed into a path.
	ErrInvalidRequest ErrorKind = "invalid_request"
	// ErrNetwork: the server could not be reached or the connection broke.
	ErrNetwork ErrorKind = "network"
	// ErrTimeout: no response within the time limit of the call.
	ErrTimeout ErrorKind = "timeout"
	// ErrCanceled: the run was canceled while the call was in progress.
	ErrCanceled ErrorKind = "canceled"
)

// Error is a call that produced no response.
type Error struct {
	Kind ErrorKind
	Err  error // the underlying error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return string(e.Kind)
	}
	return fmt.Sprintf("%s: %v", e.Kind, e.Err)
}

// Unwrap returns the underlying error, so errors.Is and errors.As see it.
func (e *Error) Unwrap() error { return e.Err }

// Errorf creates an Error of the given kind with a formatted message.
// Use %w to keep an underlying error.
func Errorf(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Err: fmt.Errorf(format, args...)}
}
