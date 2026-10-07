package validate_test

import (
	"encoding/json"
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// assertionCase checks a single assertion appended to step 1 of the probe
// scenario. want pointers are relative to that assertion ("/op", "/value").
type assertionCase struct {
	name string
	a    model.Assertion
	want []want
}

// probeAssertion is the pointer of the assertion added by an assertionCase:
// step 1 of the probe scenario already has 4 assertions.
const probeAssertion = "/scenarios/probeScenario/steps/1/expect/4"

func runAssertionCases(t *testing.T, cases []assertionCase) {
	t.Helper()
	breaks := make([]breakCase, 0, len(cases))
	for _, tc := range cases {
		wants := make([]want, len(tc.want))
		for i, w := range tc.want {
			wants[i] = want{w.code, w.sev, probeAssertion + w.ptr}
		}
		breaks = append(breaks, breakCase{
			name: tc.name,
			breaks: withScenario(func(s *model.ScenarioDef) {
				c := call(s, 1)
				c.Expect = append(c.Expect, tc.a)
			}),
			want: wants,
		})
	}
	runBreakCases(t, breaks)
}

func num(s string) json.Number { return json.Number(s) }

// Paths used by the cases.
const (
	pStatus   = "$.status"
	pDuration = "$.duration"
	pHeader   = "$.headers['content-type']"
)

// Checks of assertions: they are only checked for being well-formed here;
// they are evaluated against responses at run time. See assertions.go.
func TestAssertions(t *testing.T) {
	runAssertionCases(t, []assertionCase{
		// Valid assertions of every part of the response.
		{"status equals", model.Assertion{Path: pStatus, Op: model.OpEquals, Value: num("200")}, nil},
		{"status in", model.Assertion{Path: pStatus, Op: model.OpIn, Value: []any{num("200"), num("201")}}, nil},
		{"status range", model.Assertion{Path: pStatus, Op: model.OpGte, Value: num("200")}, nil},
		{"duration lt", model.Assertion{Path: pDuration, Op: model.OpLt, Value: num("500")}, nil},
		{"duration lt with a Go int", model.Assertion{Path: pDuration, Op: model.OpLt, Value: 500}, nil},
		{"header equals", model.Assertion{Path: pHeader, Op: model.OpEquals, Value: "application/json"}, nil},
		{"header not exists", model.Assertion{Path: "$.headers['x-debug']", Op: model.OpNotExists}, nil},
		{"header with dot notation", model.Assertion{Path: "$.headers.etag", Op: model.OpExists}, nil},
		{"all headers", model.Assertion{Path: "$.headers", Op: model.OpLength, Value: num("3")}, nil},
		{"body equals null", model.Assertion{Path: "$.body.deletedAt", Op: model.OpEquals, Value: nil}, nil},
		{"body equals object", model.Assertion{Path: "$.body.owner", Op: model.OpEquals, Value: map[string]any{"id": num("1")}}, nil},
		{"body length", model.Assertion{Path: "$.body.tags", Op: model.OpLength, Value: num("3")}, nil},
		{"body contains", model.Assertion{Path: "$.body.name", Op: model.OpContains, Value: "Pho"}, nil},
		{"body matches", model.Assertion{Path: "$.body.name", Op: model.OpMatches, Value: "^[A-Z][a-z]+$"}, nil},
		{"whole body", model.Assertion{Path: "$.body", Op: model.OpExists}, nil},
		{"body with index and quoted key", model.Assertion{Path: "$.body.items[0]['first-name']", Op: model.OpExists}, nil},
		{"body array element", model.Assertion{Path: "$.body[0].id", Op: model.OpGt, Value: num("0")}, nil},

		// Templates are not type-checked: the type is known at run time.
		{"template for in", model.Assertion{Path: "$.body.state", Op: model.OpIn, Value: "{{allowedStates}}"}, nil},
		{"template for lt", model.Assertion{Path: "$.body.price", Op: model.OpLt, Value: "{{maxPrice}}"}, nil},
		{"template for matches", model.Assertion{Path: "$.body.name", Op: model.OpMatches, Value: "{{pattern}}"}, nil},

		// Path.
		{"missing path", model.Assertion{Op: model.OpEquals, Value: num("200")},
			[]want{{validate.V0730, diag.Error, "/path"}}},
		{"path without root", model.Assertion{Path: "status", Op: model.OpEquals, Value: num("200")},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"malformed path", model.Assertion{Path: "$.body.items[", Op: model.OpExists},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"path with a wildcard", model.Assertion{Path: "$.body.items[*].id", Op: model.OpExists},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"path with a filter", model.Assertion{Path: "$.body.items[?(@.price > 10)]", Op: model.OpExists},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"path to the whole response", model.Assertion{Path: "$", Op: model.OpExists},
			[]want{{validate.V0725, diag.Error, "/path"}}},
		{"path to an unknown part", model.Assertion{Path: "$.cookies.session", Op: model.OpExists},
			[]want{{validate.V0725, diag.Error, "/path"}}},
		{"path from the body, not from the response", model.Assertion{Path: "$.name", Op: model.OpEquals, Value: "Phone"},
			[]want{{validate.V0725, diag.Error, "/path"}}},
		{"parts are case-sensitive", model.Assertion{Path: "$.Body.name", Op: model.OpExists},
			[]want{{validate.V0725, diag.Error, "/path"}}},
		{"path starting with an index", model.Assertion{Path: "$[0]", Op: model.OpExists},
			[]want{{validate.V0725, diag.Error, "/path"}}},
		{"path inside the status", model.Assertion{Path: "$.status.code", Op: model.OpEquals, Value: num("200")},
			[]want{{validate.V0729, diag.Error, "/path"}}},
		{"path inside the duration", model.Assertion{Path: "$.duration[0]", Op: model.OpLt, Value: num("100")},
			[]want{{validate.V0729, diag.Error, "/path"}}},
		{"path inside a header", model.Assertion{Path: "$.headers['content-type'].charset", Op: model.OpExists},
			[]want{{validate.V0729, diag.Error, "/path"}}},
		{"header by index", model.Assertion{Path: "$.headers[0]", Op: model.OpExists},
			[]want{{validate.V0729, diag.Error, "/path"}}},
		{"header name with upper case", model.Assertion{Path: "$.headers['Content-Type']", Op: model.OpExists},
			[]want{{validate.V0737, diag.Warning, "/path"}}},

		// Operator.
		{"missing operator", model.Assertion{Path: pStatus, Value: num("200")},
			[]want{{validate.V0726, diag.Error, "/op"}}},
		{"unsupported operator", model.Assertion{Path: pStatus, Op: "between", Value: []any{num("200"), num("299")}},
			[]want{{validate.V0727, diag.Error, "/op"}}},
		{"operator names are case-sensitive", model.Assertion{Path: pStatus, Op: "Equals", Value: num("200")},
			[]want{{validate.V0727, diag.Error, "/op"}}},
		{
			// Applicability is not checked when either side is unusable.
			"bad path and unknown operator", model.Assertion{Path: "$.cookies", Op: "between"},
			[]want{
				{validate.V0725, diag.Error, "/path"},
				{validate.V0727, diag.Error, "/op"},
			},
		},
		{
			// The value is still checked against a known operator.
			"bad path, known operator without value", model.Assertion{Path: "$.cookies", Op: model.OpLt},
			[]want{
				{validate.V0725, diag.Error, "/path"},
				{validate.V0732, diag.Error, "/value"},
			},
		},

		// Applicability of operators to parts of the response.
		{"status matches", model.Assertion{Path: pStatus, Op: model.OpMatches, Value: "2.."},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"status exists", model.Assertion{Path: pStatus, Op: model.OpExists},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"status length", model.Assertion{Path: pStatus, Op: model.OpLength, Value: num("3")},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"duration equals", model.Assertion{Path: pDuration, Op: model.OpEquals, Value: num("100")},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"header length", model.Assertion{Path: "$.headers.etag", Op: model.OpLength, Value: num("32")},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"header lt", model.Assertion{Path: "$.headers['x-count']", Op: model.OpLt, Value: num("10")},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"any operator fits a body value", model.Assertion{Path: "$.body.count", Op: model.OpLte, Value: num("10")}, nil},

		// Value presence.
		{"in without value", model.Assertion{Path: pStatus, Op: model.OpIn},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"lt without value", model.Assertion{Path: pDuration, Op: model.OpLt},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"contains without value", model.Assertion{Path: "$.body.name", Op: model.OpContains},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"matches without value", model.Assertion{Path: "$.body.name", Op: model.OpMatches},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"length without value", model.Assertion{Path: "$.body.tags", Op: model.OpLength},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"exists with value", model.Assertion{Path: "$.body.id", Op: model.OpExists, Value: true},
			[]want{{validate.V0733, diag.Warning, "/value"}}},
		{"notExists with value", model.Assertion{Path: "$.headers['x-debug']", Op: model.OpNotExists, Value: "1"},
			[]want{{validate.V0733, diag.Warning, "/value"}}},

		// Value types.
		{"in with a string", model.Assertion{Path: pStatus, Op: model.OpIn, Value: "200,201"},
			[]want{{validate.V0734, diag.Error, "/value"}}},
		{"in with an object", model.Assertion{Path: "$.body.state", Op: model.OpIn, Value: map[string]any{"a": true}},
			[]want{{validate.V0734, diag.Error, "/value"}}},
		{"matches with a number", model.Assertion{Path: "$.body.code", Op: model.OpMatches, Value: num("42")},
			[]want{{validate.V0735, diag.Error, "/value"}}},
		{"matches with an invalid regexp", model.Assertion{Path: "$.body.name", Op: model.OpMatches, Value: "(unclosed"},
			[]want{{validate.V0735, diag.Error, "/value"}}},
		{"lt with a string", model.Assertion{Path: pDuration, Op: model.OpLt, Value: "fast"},
			[]want{{validate.V0736, diag.Error, "/value"}}},
		{"gt with a bool", model.Assertion{Path: "$.body.price", Op: model.OpGt, Value: true},
			[]want{{validate.V0736, diag.Error, "/value"}}},
		{"length with a string", model.Assertion{Path: "$.body.tags", Op: model.OpLength, Value: "three"},
			[]want{{validate.V0736, diag.Error, "/value"}}},
		{"lt with a malformed json.Number", model.Assertion{Path: pDuration, Op: model.OpLt, Value: num("1e")},
			[]want{{validate.V0736, diag.Error, "/value"}}},

		// Independent problems are all reported.
		{
			"path and value problems together", model.Assertion{Path: "$.body.price[", Op: model.OpLt, Value: "cheap"},
			[]want{
				{validate.V0731, diag.Error, "/path"},
				{validate.V0736, diag.Error, "/value"},
			},
		},
		{
			"header case warning and operator error", model.Assertion{Path: "$.headers['X-Count']", Op: model.OpGt, Value: num("1")},
			[]want{
				{validate.V0737, diag.Warning, "/path"},
				{validate.V0728, diag.Error, "/op"},
			},
		},
	})
}
