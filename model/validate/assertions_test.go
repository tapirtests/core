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

// Checks of assertions: they are only checked for being well-formed here;
// they are evaluated against responses at run time. See assertions.go.
func TestAssertions(t *testing.T) {
	runAssertionCases(t, []assertionCase{
		// Valid assertions of every target.
		{"status equals", model.Assertion{Target: model.TargetStatus, Op: model.OpEquals, Value: num("200")}, nil},
		{"status in", model.Assertion{Target: model.TargetStatus, Op: model.OpIn, Value: []any{num("200"), num("201")}}, nil},
		{"status range", model.Assertion{Target: model.TargetStatus, Op: model.OpGte, Value: num("200")}, nil},
		{"duration lt", model.Assertion{Target: model.TargetDuration, Op: model.OpLt, Value: num("500")}, nil},
		{"duration lt with a Go int", model.Assertion{Target: model.TargetDuration, Op: model.OpLt, Value: 500}, nil},
		{"header equals", model.Assertion{Target: model.TargetHeader, Path: "Content-Type", Op: model.OpEquals, Value: "application/json"}, nil},
		{"header not exists", model.Assertion{Target: model.TargetHeader, Path: "X-Debug", Op: model.OpNotExists}, nil},
		{"body equals null", model.Assertion{Target: model.TargetBody, Path: "$.deletedAt", Op: model.OpEquals, Value: nil}, nil},
		{"body equals object", model.Assertion{Target: model.TargetBody, Path: "$.owner", Op: model.OpEquals, Value: map[string]any{"id": num("1")}}, nil},
		{"body length", model.Assertion{Target: model.TargetBody, Path: "$.tags", Op: model.OpLength, Value: num("3")}, nil},
		{"body contains", model.Assertion{Target: model.TargetBody, Path: "$.name", Op: model.OpContains, Value: "Pho"}, nil},
		{"body matches", model.Assertion{Target: model.TargetBody, Path: "$.name", Op: model.OpMatches, Value: "^[A-Z][a-z]+$"}, nil},

		// Templates are not type-checked: the type is known at run time.
		{"template for in", model.Assertion{Target: model.TargetBody, Path: "$.state", Op: model.OpIn, Value: "{{allowedStates}}"}, nil},
		{"template for lt", model.Assertion{Target: model.TargetBody, Path: "$.price", Op: model.OpLt, Value: "{{maxPrice}}"}, nil},
		{"template for matches", model.Assertion{Target: model.TargetBody, Path: "$.name", Op: model.OpMatches, Value: "{{pattern}}"}, nil},

		// Target and operator.
		{"missing target", model.Assertion{Op: model.OpEquals, Value: num("200")},
			[]want{{validate.V0724, diag.Error, "/target"}}},
		{"unsupported target", model.Assertion{Target: "cookie", Path: "session", Op: model.OpExists},
			[]want{{validate.V0725, diag.Error, "/target"}}},
		{"missing operator", model.Assertion{Target: model.TargetStatus, Value: num("200")},
			[]want{{validate.V0726, diag.Error, "/op"}}},
		{"unsupported operator", model.Assertion{Target: model.TargetStatus, Op: "between", Value: []any{num("200"), num("299")}},
			[]want{{validate.V0727, diag.Error, "/op"}}},
		{"operator names are case-sensitive", model.Assertion{Target: model.TargetStatus, Op: "Equals", Value: num("200")},
			[]want{{validate.V0727, diag.Error, "/op"}}},
		{
			// Applicability is not checked when either side is unknown.
			"unknown target and operator", model.Assertion{Target: "cookie", Op: "between"},
			[]want{
				{validate.V0725, diag.Error, "/target"},
				{validate.V0727, diag.Error, "/op"},
			},
		},
		{
			// The value is still checked against a known operator.
			"unknown target, known operator without value", model.Assertion{Target: "cookie", Op: model.OpLt},
			[]want{
				{validate.V0725, diag.Error, "/target"},
				{validate.V0732, diag.Error, "/value"},
			},
		},

		// Applicability of operators to targets.
		{"status matches", model.Assertion{Target: model.TargetStatus, Op: model.OpMatches, Value: "2.."},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"status exists", model.Assertion{Target: model.TargetStatus, Op: model.OpExists},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"duration equals", model.Assertion{Target: model.TargetDuration, Op: model.OpEquals, Value: num("100")},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"header length", model.Assertion{Target: model.TargetHeader, Path: "ETag", Op: model.OpLength, Value: num("32")},
			[]want{{validate.V0728, diag.Error, "/op"}}},
		{"header lt", model.Assertion{Target: model.TargetHeader, Path: "X-Count", Op: model.OpLt, Value: num("10")},
			[]want{{validate.V0728, diag.Error, "/op"}}},

		// Path.
		{"status with path", model.Assertion{Target: model.TargetStatus, Path: "$.code", Op: model.OpEquals, Value: num("200")},
			[]want{{validate.V0729, diag.Error, "/path"}}},
		{"duration with path", model.Assertion{Target: model.TargetDuration, Path: "total", Op: model.OpLt, Value: num("100")},
			[]want{{validate.V0729, diag.Error, "/path"}}},
		{"body without path", model.Assertion{Target: model.TargetBody, Op: model.OpExists},
			[]want{{validate.V0730, diag.Error, "/path"}}},
		{"header without name", model.Assertion{Target: model.TargetHeader, Op: model.OpExists},
			[]want{{validate.V0730, diag.Error, "/path"}}},
		{"body path without $", model.Assertion{Target: model.TargetBody, Path: "name", Op: model.OpExists},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"body path with a wildcard", model.Assertion{Target: model.TargetBody, Path: "$.items[*].id", Op: model.OpExists},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"malformed body path", model.Assertion{Target: model.TargetBody, Path: "$.items[", Op: model.OpExists},
			[]want{{validate.V0731, diag.Error, "/path"}}},
		{"body path to the whole body", model.Assertion{Target: model.TargetBody, Path: "$", Op: model.OpExists}, nil},
		{"body path with index and quoted key", model.Assertion{Target: model.TargetBody, Path: "$.items[0]['first-name']", Op: model.OpExists}, nil},
		{"header name is not a JSONPath", model.Assertion{Target: model.TargetHeader, Path: "$.Content-Type", Op: model.OpExists}, nil},

		// Value presence.
		{"in without value", model.Assertion{Target: model.TargetStatus, Op: model.OpIn},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"lt without value", model.Assertion{Target: model.TargetDuration, Op: model.OpLt},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"contains without value", model.Assertion{Target: model.TargetBody, Path: "$.name", Op: model.OpContains},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"matches without value", model.Assertion{Target: model.TargetBody, Path: "$.name", Op: model.OpMatches},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"length without value", model.Assertion{Target: model.TargetBody, Path: "$.tags", Op: model.OpLength},
			[]want{{validate.V0732, diag.Error, "/value"}}},
		{"exists with value", model.Assertion{Target: model.TargetBody, Path: "$.id", Op: model.OpExists, Value: true},
			[]want{{validate.V0733, diag.Warning, "/value"}}},
		{"notExists with value", model.Assertion{Target: model.TargetHeader, Path: "X-Debug", Op: model.OpNotExists, Value: "1"},
			[]want{{validate.V0733, diag.Warning, "/value"}}},

		// Value types.
		{"in with a string", model.Assertion{Target: model.TargetStatus, Op: model.OpIn, Value: "200,201"},
			[]want{{validate.V0734, diag.Error, "/value"}}},
		{"in with an object", model.Assertion{Target: model.TargetBody, Path: "$.state", Op: model.OpIn, Value: map[string]any{"a": true}},
			[]want{{validate.V0734, diag.Error, "/value"}}},
		{"matches with a number", model.Assertion{Target: model.TargetBody, Path: "$.code", Op: model.OpMatches, Value: num("42")},
			[]want{{validate.V0735, diag.Error, "/value"}}},
		{"matches with an invalid regexp", model.Assertion{Target: model.TargetBody, Path: "$.name", Op: model.OpMatches, Value: "(unclosed"},
			[]want{{validate.V0735, diag.Error, "/value"}}},
		{"lt with a string", model.Assertion{Target: model.TargetDuration, Op: model.OpLt, Value: "fast"},
			[]want{{validate.V0736, diag.Error, "/value"}}},
		{"gt with a bool", model.Assertion{Target: model.TargetBody, Path: "$.price", Op: model.OpGt, Value: true},
			[]want{{validate.V0736, diag.Error, "/value"}}},
		{"length with a string", model.Assertion{Target: model.TargetBody, Path: "$.tags", Op: model.OpLength, Value: "three"},
			[]want{{validate.V0736, diag.Error, "/value"}}},
		{"lt with a malformed json.Number", model.Assertion{Target: model.TargetDuration, Op: model.OpLt, Value: num("1e")},
			[]want{{validate.V0736, diag.Error, "/value"}}},

		// Independent problems are all reported.
		{
			"path and value problems together", model.Assertion{Target: model.TargetBody, Path: "price", Op: model.OpLt, Value: "cheap"},
			[]want{
				{validate.V0731, diag.Error, "/path"},
				{validate.V0736, diag.Error, "/value"},
			},
		},
	})
}
