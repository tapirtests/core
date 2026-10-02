package validate_test

import (
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// probe is a valid request that no scenario uses. Request cases break it
// instead of the requests of the reference project, so that later checks of
// scenario steps do not add diagnostics to these cases.
//
// Inputs: 0 auth.bearer, 1 path.id, 2 query.q, 3 header.X-Trace, 4 body.name.
func probe() *model.RequestDef {
	return &model.RequestDef{
		ID:        "probe",
		Operation: model.HTTPOperation{Method: "POST", Path: "/things/{id}", Consumes: "application/json"},
		Inputs: []model.InputField{
			{Name: "auth.bearer", Required: true},
			{Name: "path.id", Required: true, Schema: model.Schema(`{"type":"integer"}`)},
			{Name: "query.q"},
			{Name: "header.X-Trace"},
			{Name: "body.name", Required: true, Schema: model.Schema(`{"type":"string"}`)},
		},
		Responses: map[string]model.ResponseDef{
			"200":     {Description: "OK", Schema: model.Schema(`{"type":"object"}`)},
			"default": {Description: "Error"},
		},
	}
}

// withProbe adds the probe request, lets breaks modify it and returns the
// break function for a breakCase.
func withProbe(breaks func(r *model.RequestDef)) func(p *model.Project) {
	return func(p *model.Project) {
		r := probe()
		breaks(r)
		p.Requests[r.ID] = r
	}
}

// otherProtocol is an Operation of a protocol the core does not support.
type otherProtocol struct{}

func (otherProtocol) Protocol() model.Protocol { return "graphql" }

// Protocol-independent checks of request definitions; see requests.go.
// HTTP-specific checks are in requests_http_test.go.
func TestRequests(t *testing.T) {
	runBreakCases(t, []breakCase{
		// Valid variations.
		{
			name:   "probe request is valid",
			breaks: withProbe(func(r *model.RequestDef) {}),
			want:   nil,
		},
		{
			name: "operation as a pointer is valid",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = &model.HTTPOperation{Method: "POST", Path: "/things/{id}"}
			}),
			want: nil,
		},
		{
			name: "request without operationId keyed by METHOD /path",
			breaks: func(p *model.Project) {
				p.Requests["GET /health"] = &model.RequestDef{
					ID:        "GET /health",
					Operation: model.HTTPOperation{Method: "GET", Path: "/health"},
				}
			},
			want: nil,
		},

		// The definition itself.
		{
			name:   "nil request",
			breaks: func(p *model.Project) { p.Requests["broken"] = nil },
			want:   []want{{validate.V0600, diag.Error, "/requests/broken"}},
		},
		{
			name: "missing request ID",
			breaks: func(p *model.Project) {
				r := probe()
				r.ID = ""
				p.Requests["probe"] = r
			},
			want: []want{{validate.V0601, diag.Error, "/requests/probe/id"}},
		},
		{
			name: "request ID differs from key",
			breaks: func(p *model.Project) {
				r := probe()
				r.ID = "other"
				p.Requests["probe"] = r
			},
			want: []want{{validate.V0602, diag.Error, "/requests/probe/id"}},
		},
		{
			name: "key with a slash is escaped in the pointer",
			breaks: func(p *model.Project) {
				p.Requests["GET /health"] = &model.RequestDef{ID: "GET /health"}
			},
			want: []want{{validate.V0603, diag.Error, "/requests/GET ~1health/operation"}},
		},
		{
			// Without an operation, protocol-specific input checks are skipped.
			name:   "missing operation",
			breaks: withProbe(func(r *model.RequestDef) { r.Operation = nil }),
			want:   []want{{validate.V0603, diag.Error, "/requests/probe/operation"}},
		},
		{
			name:   "nil operation pointer",
			breaks: withProbe(func(r *model.RequestDef) { r.Operation = (*model.HTTPOperation)(nil) }),
			want:   []want{{validate.V0603, diag.Error, "/requests/probe/operation"}},
		},
		{
			name:   "unsupported protocol",
			breaks: withProbe(func(r *model.RequestDef) { r.Operation = otherProtocol{} }),
			want:   []want{{validate.V0604, diag.Error, "/requests/probe/operation"}},
		},
		{
			// Common input checks still apply when the protocol is unknown.
			name: "unsupported protocol keeps common input checks",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = otherProtocol{}
				r.Inputs[2].Name = ""
			}),
			want: []want{
				{validate.V0604, diag.Error, "/requests/probe/operation"},
				{validate.V0611, diag.Error, "/requests/probe/inputs/2/name"},
			},
		},

		// Input contract: checks common to all protocols.
		{
			name:   "missing input name",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[2].Name = "" }),
			want:   []want{{validate.V0611, diag.Error, "/requests/probe/inputs/2/name"}},
		},
		{
			name: "duplicate input is reported on the repeat only",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Inputs = append(r.Inputs, model.InputField{Name: "query.q"})
			}),
			want: []want{{validate.V0614, diag.Error, "/requests/probe/inputs/5/name"}},
		},
		{
			name:   "auth input with unknown scheme",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[0].Name = "auth.jwt" }),
			want:   []want{{validate.V0617, diag.Error, "/requests/probe/inputs/0/name"}},
		},
		{
			name:   "auth input without scheme",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[0].Name = "auth." }),
			want:   []want{{validate.V0613, diag.Error, "/requests/probe/inputs/0/name"}},
		},
		{
			// Every auth.bearer input of the reference project is reported too.
			name:   "auth inputs when the project has no schemes",
			breaks: func(p *model.Project) { p.SecuritySchemes = nil },
			want: []want{
				{validate.V0617, diag.Error, "/requests/createProduct/inputs/0/name"},
				{validate.V0617, diag.Error, "/requests/deleteProduct/inputs/0/name"},
				{validate.V0617, diag.Error, "/requests/getProduct/inputs/0/name"},
			},
		},

		// Output contract and schemas.
		{
			name: "response key with a range",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Responses["2XX"] = model.ResponseDef{}
			}),
			want: []want{{validate.V0619, diag.Error, "/requests/probe/responses/2XX"}},
		},
		{
			name: "response key out of range",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Responses["600"] = model.ResponseDef{}
			}),
			want: []want{{validate.V0619, diag.Error, "/requests/probe/responses/600"}},
		},
		{
			name: "response key with a leading zero",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Responses["0200"] = model.ResponseDef{}
			}),
			want: []want{{validate.V0619, diag.Error, "/requests/probe/responses/0200"}},
		},
		{
			name: "invalid response schema",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Responses["200"] = model.ResponseDef{Schema: model.Schema(`{"type":`)}
			}),
			want: []want{{validate.V0620, diag.Error, "/requests/probe/responses/200/schema"}},
		},
		{
			name:   "invalid input schema",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[1].Schema = model.Schema(`{type: integer}`) }),
			want:   []want{{validate.V0620, diag.Error, "/requests/probe/inputs/1/schema"}},
		},
	})
}
