package validate_test

import (
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// HTTP-specific checks of request definitions; see requests_http.go.
// The cases break the probe request from requests_test.go.
func TestHTTPRequests(t *testing.T) {
	runBreakCases(t, []breakCase{
		// Valid variations.
		{
			name: "several path parameters",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "GET", Path: "/shops/{shopId}/things/{id}"}
				r.Inputs = append(r.Inputs, model.InputField{Name: "path.shopId", Required: true})
			}),
			want: nil,
		},
		{
			name:   "whole body input alone is valid",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[4].Name = "body" }),
			want:   nil,
		},

		// Method.
		{
			name: "missing HTTP method",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Path: "/things/{id}"}
			}),
			want: []want{{validate.V0605, diag.Error, "/requests/probe/operation/method"}},
		},
		{
			name: "lower-case HTTP method",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "post", Path: "/things/{id}"}
			}),
			want: []want{{validate.V0606, diag.Error, "/requests/probe/operation/method"}},
		},
		{
			name: "unsupported HTTP method",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "TRACE", Path: "/things/{id}"}
			}),
			want: []want{{validate.V0606, diag.Error, "/requests/probe/operation/method"}},
		},

		// Path template.
		{
			name: "missing HTTP path does not cascade to path inputs",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST"}
			}),
			want: []want{{validate.V0607, diag.Error, "/requests/probe/operation/path"}},
		},
		{
			name: "HTTP path without leading slash",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST", Path: "things/{id}"}
			}),
			want: []want{{validate.V0608, diag.Error, "/requests/probe/operation/path"}},
		},
		{
			name: "unclosed brace does not cascade to path inputs",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST", Path: "/things/{id"}
			}),
			want: []want{{validate.V0609, diag.Error, "/requests/probe/operation/path"}},
		},
		{
			name: "unexpected closing brace",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST", Path: "/things/id}"}
			}),
			want: []want{{validate.V0609, diag.Error, "/requests/probe/operation/path"}},
		},
		{
			name: "empty path parameter",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST", Path: "/things/{}"}
			}),
			want: []want{{validate.V0609, diag.Error, "/requests/probe/operation/path"}},
		},
		{
			name: "nested braces",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST", Path: "/things/{a{b}}"}
			}),
			want: []want{{validate.V0609, diag.Error, "/requests/probe/operation/path"}},
		},
		{
			name: "path parameter without input",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "POST", Path: "/shops/{shopId}/things/{id}"}
			}),
			want: []want{{validate.V0610, diag.Error, "/requests/probe/inputs"}},
		},

		// Input locations.
		{
			name:   "unknown input location",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[4].Name = "json.name" }),
			want:   []want{{validate.V0612, diag.Error, "/requests/probe/inputs/4/name"}},
		},
		{
			name:   "location is case-sensitive",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[2].Name = "Query.q" }),
			want:   []want{{validate.V0612, diag.Error, "/requests/probe/inputs/2/name"}},
		},
		{
			name:   "input without field after dot",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[2].Name = "query." }),
			want:   []want{{validate.V0613, diag.Error, "/requests/probe/inputs/2/name"}},
		},
		{
			name:   "input with location only",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[2].Name = "query" }),
			want:   []want{{validate.V0613, diag.Error, "/requests/probe/inputs/2/name"}},
		},

		// Duplicates.
		{
			name: "header inputs differing only in case are duplicates",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Inputs = append(r.Inputs, model.InputField{Name: "header.x-trace"})
			}),
			want: []want{{validate.V0614, diag.Error, "/requests/probe/inputs/5/name"}},
		},
		{
			name: "query inputs differing only in case are different",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Inputs = append(r.Inputs, model.InputField{Name: "query.Q"})
			}),
			want: nil,
		},

		// Path inputs.
		{
			name: "path input without template parameter",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Inputs = append(r.Inputs, model.InputField{Name: "path.shopId", Required: true})
			}),
			want: []want{{validate.V0615, diag.Error, "/requests/probe/inputs/5/name"}},
		},
		{
			name:   "optional path input",
			breaks: withProbe(func(r *model.RequestDef) { r.Inputs[1].Required = false }),
			want:   []want{{validate.V0616, diag.Error, "/requests/probe/inputs/1/required"}},
		},

		// Body.
		{
			name: "whole body combined with body fields",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Inputs = append(r.Inputs, model.InputField{Name: "body"})
			}),
			want: []want{{validate.V0618, diag.Error, "/requests/probe/inputs/5/name"}},
		},

		// Several independent problems are all reported.
		{
			name: "problems in operation and inputs",
			breaks: withProbe(func(r *model.RequestDef) {
				r.Operation = model.HTTPOperation{Method: "post", Path: "/things/{id}"}
				r.Inputs[1].Required = false
				r.Inputs[2].Name = "json.q"
			}),
			want: []want{
				{validate.V0606, diag.Error, "/requests/probe/operation/method"},
				{validate.V0616, diag.Error, "/requests/probe/inputs/1/required"},
				{validate.V0612, diag.Error, "/requests/probe/inputs/2/name"},
			},
		},
	})
}
