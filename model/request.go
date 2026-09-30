package model

import (
	"encoding/json"
	"time"
)

// RequestDef is the definition of a single API request, stored in the
// Requests bucket. Most definitions are imported from the API specification;
// users may add custom ones with the same strict contract.
//
// The contract is fixed: a call may pass only the declared inputs, and the
// result is described by Responses. Protocol-specific details (method, path)
// live in Operation. Expectations are not part of the definition — they are
// set by each RequestCall.
type RequestDef struct {
	ID          RequestID              // identifier
	Operation   Operation              // http, graphql, soap, ...
	Inputs      []InputField           // input fields (from Swagger, OpenAPI, ...)
	Responses   map[string]ResponseDef // outcome fields
	Custom      bool                   // Request created by a user
	Description string                 // description
}

// RequestCall is a step of a scenario that calls a RequestDef.
//
// It is the usage of a request: it passes input values taken from the
// scenario scope, checks the result against Expect and saves the selected
// parts of the response (Extract) back to the scenario scope. The full
// response still goes to the report regardless of what is extracted.
type RequestCall struct {
	ID          string            // request caller identifier
	RequestID   RequestID         // request identifier
	Inputs      map[string]Value  // fields, that pass into the request
	Extract     map[string]string // fields, that extract from the request
	Expect      []Assertion       // assertion rules
	IgnoreError bool              // ignore failed request
	Timeout     time.Duration     // maximum request time
	Description string            // description
}

// StepID implements Step.
func (c *RequestCall) StepID() string { return c.ID }

func (*RequestCall) isStep() {}

// ResponseDef describes one possible response of a request as documented by
// the API specification. It is the output contract, not an actual response.
type ResponseDef struct {
	Description string // description
	Schema      Schema // schema of the response body from API docs
}

// InputField is one input of a RequestDef contract.
//
// The name encodes where the value goes: path.id, query.q, header.X-Trace,
// cookie.session, body.price, auth.bearer. Drivers map these names onto the
// actual request, so the contract itself stays protocol-independent.
type InputField struct {
	Name        string // body.price
	Required    bool   // a call must pass this input (an explicit null counts as passed)
	Schema      Schema // value type
	Description string // description
}

// Schema is a raw JSON Schema document. The model does not interpret it;
// it is used by response validation, the validator and the generator.
// An empty Schema means the type is not documented.
type Schema json.RawMessage

// RequestID identifies a RequestDef: the operationId from the specification,
// or "METHOD /path" when the operation has none.
type RequestID string

// String implements fmt.Stringer.
func (r RequestID) String() string {
	return string(r)
}
