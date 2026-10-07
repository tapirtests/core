package validate_test

import (
	"encoding/json"
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// probeScenarioKey is the key of the probe scenario in the Scenario bucket.
const probeScenarioKey = "probeScenario"

// probeScenario is a valid scenario that no group calls. Scenario cases break
// it instead of the scenarios of the reference project, so that later checks
// of groups do not add diagnostics to these cases. It uses the requests of
// the reference project.
//
// Inputs:  0 token (no default: must be passed), 1 name (default "Phone").
// Outputs: 0 productId.
// Steps:
//
//	0 "create": createProduct, extracts productId, 2 assertions
//	1 "get":    getProduct with an explicit null auth, 4 assertions
func probeScenario() *model.ScenarioDef {
	return &model.ScenarioDef{
		ID:   probeScenarioKey,
		Name: "Probe",
		Inputs: []model.Param{
			{Name: "token"},
			{Name: "name", HasDefault: true, Default: "Phone"},
		},
		Outputs: []model.Output{{Name: "productId", Value: "{{productId}}"}},
		Steps: []model.Step{
			&model.RequestCall{
				ID:        "create",
				RequestID: "createProduct",
				Inputs: map[string]model.Value{
					"auth.bearer": "{{token}}",
					"body.name":   "{{name}}",
					"body.price":  json.Number("10"),
				},
				Extract: map[string]string{"productId": "$.body.id"},
				Expect: []model.Assertion{
					{Path: "$.status", Op: model.OpEquals, Value: json.Number("201")},
					{Path: "$.body.name", Op: model.OpEquals, Value: "{{name}}"},
				},
			},
			&model.RequestCall{
				ID:        "get",
				RequestID: "getProduct",
				Inputs: map[string]model.Value{
					"auth.bearer": nil, // explicit null: intentionally no authorization
					"path.id":     "{{productId}}",
				},
				Expect: []model.Assertion{
					{Path: "$.status", Op: model.OpIn, Value: []any{json.Number("401"), json.Number("403")}},
					{Path: "$.headers['content-type']", Op: model.OpMatches, Value: "json"},
					{Path: "$.duration", Op: model.OpLt, Value: json.Number("1000")},
					{Path: "$.body.message", Op: model.OpExists},
				},
			},
		},
	}
}

// withScenario adds the probe scenario under probeScenarioKey, lets breaks
// modify it and returns the break function for a breakCase.
func withScenario(breaks func(s *model.ScenarioDef)) func(p *model.Project) {
	return func(p *model.Project) {
		s := probeScenario()
		breaks(s)
		p.Scenarios[probeScenarioKey] = s
	}
}

// call returns step i of s as a request call.
func call(s *model.ScenarioDef, i int) *model.RequestCall {
	return s.Steps[i].(*model.RequestCall)
}

// Checks of scenario definitions: the definition itself, the signature and
// the list of steps; see scenarios.go.
func TestScenarios(t *testing.T) {
	runBreakCases(t, []breakCase{
		// Valid variations.
		{
			name:   "probe scenario is valid",
			breaks: withScenario(func(s *model.ScenarioDef) {}),
			want:   nil,
		},
		{
			name:   "input and output may share a name",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Outputs[0].Name = "token" }),
			want:   nil,
		},
		{
			name: "scenario without inputs and outputs",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Inputs, s.Outputs = nil, nil
				call(s, 0).Inputs["auth.bearer"] = "static"
				call(s, 0).Inputs["body.name"] = "Phone"
			}),
			want: nil,
		},

		// The definition itself.
		{
			name:   "nil scenario",
			breaks: func(p *model.Project) { p.Scenarios["broken"] = nil },
			want:   []want{{validate.V0700, diag.Error, "/scenarios/broken"}},
		},
		{
			name:   "missing scenario ID",
			breaks: withScenario(func(s *model.ScenarioDef) { s.ID = "" }),
			want:   []want{{validate.V0701, diag.Error, "/scenarios/probeScenario/id"}},
		},
		{
			name:   "scenario ID differs from key",
			breaks: withScenario(func(s *model.ScenarioDef) { s.ID = "other" }),
			want:   []want{{validate.V0702, diag.Error, "/scenarios/probeScenario/id"}},
		},

		// Inputs.
		{
			name:   "missing input name",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Name = "" }),
			want:   []want{{validate.V0703, diag.Error, "/scenarios/probeScenario/inputs/1/name"}},
		},
		{
			// Dots are reserved for automatic output names "<alias>.<output>".
			name:   "input name with a dot",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Name = "product.name" }),
			want:   []want{{validate.V0704, diag.Error, "/scenarios/probeScenario/inputs/1/name"}},
		},
		{
			name:   "input name with a dash",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Name = "product-name" }),
			want:   []want{{validate.V0704, diag.Error, "/scenarios/probeScenario/inputs/1/name"}},
		},
		{
			name: "duplicate input is reported on the repeat only",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Inputs = append(s.Inputs, model.Param{Name: "token"})
			}),
			want: []want{{validate.V0705, diag.Error, "/scenarios/probeScenario/inputs/2/name"}},
		},
		{
			// Null is a valid default and differs from "no default".
			name: "null default is valid",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Inputs[1] = model.Param{Name: "name", HasDefault: true, Default: nil}
			}),
			want: nil,
		},
		{
			name:   "default value that is not marked as set",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[0].Default = "x" }),
			want:   []want{{validate.V0710, diag.Error, "/scenarios/probeScenario/inputs/0/default"}},
		},
		{
			// The default is not checked for a duplicate: it would only add
			// to a problem of the declaration.
			name: "duplicate input with a broken default",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Inputs = append(s.Inputs, model.Param{Name: "token", Default: "x"})
			}),
			want: []want{{validate.V0705, diag.Error, "/scenarios/probeScenario/inputs/2/name"}},
		},

		// Outputs.
		{
			name:   "missing output name",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Outputs[0].Name = "" }),
			want:   []want{{validate.V0706, diag.Error, "/scenarios/probeScenario/outputs/0/name"}},
		},
		{
			name:   "invalid output name",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Outputs[0].Name = "product.id" }),
			want:   []want{{validate.V0707, diag.Error, "/scenarios/probeScenario/outputs/0/name"}},
		},
		{
			name: "duplicate output",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Outputs = append(s.Outputs, model.Output{Name: "productId", Value: "{{productId}}"})
			}),
			want: []want{{validate.V0708, diag.Error, "/scenarios/probeScenario/outputs/1/name"}},
		},
		{
			name:   "output without value",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Outputs[0].Value = nil }),
			want:   []want{{validate.V0709, diag.Error, "/scenarios/probeScenario/outputs/0/value"}},
		},
		{
			// An invalid but unique name does not stop further checks.
			name: "invalid output name without value",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Outputs[0] = model.Output{Name: "product-id"}
			}),
			want: []want{
				{validate.V0707, diag.Error, "/scenarios/probeScenario/outputs/0/name"},
				{validate.V0709, diag.Error, "/scenarios/probeScenario/outputs/0/value"},
			},
		},
		{
			name: "duplicate output without value",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Outputs = append(s.Outputs, model.Output{Name: "productId"})
			}),
			want: []want{{validate.V0708, diag.Error, "/scenarios/probeScenario/outputs/1/name"}},
		},

		// Steps.
		{
			name:   "scenario without steps",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Steps = nil }),
			want:   []want{{validate.V0711, diag.Warning, "/scenarios/probeScenario/steps"}},
		},
		{
			name:   "nil step",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Steps = append(s.Steps, nil) }),
			want:   []want{{validate.V0712, diag.Error, "/scenarios/probeScenario/steps/2"}},
		},
		{
			name: "nil request call",
			breaks: withScenario(func(s *model.ScenarioDef) {
				s.Steps = append(s.Steps, (*model.RequestCall)(nil))
			}),
			want: []want{{validate.V0712, diag.Error, "/scenarios/probeScenario/steps/2"}},
		},
		{
			name:   "missing step ID",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).ID = "" }),
			want:   []want{{validate.V0713, diag.Error, "/scenarios/probeScenario/steps/0/id"}},
		},
		{
			name:   "duplicate step ID is reported on the repeat only",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 1).ID = "create" }),
			want:   []want{{validate.V0714, diag.Error, "/scenarios/probeScenario/steps/1/id"}},
		},
		{
			name: "empty step IDs are not reported as duplicates",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).ID = ""
				call(s, 1).ID = ""
			}),
			want: []want{
				{validate.V0713, diag.Error, "/scenarios/probeScenario/steps/0/id"},
				{validate.V0713, diag.Error, "/scenarios/probeScenario/steps/1/id"},
			},
		},
		{
			name: "step IDs are unique per scenario, not per project",
			breaks: withScenario(func(s *model.ScenarioDef) {
				// "create" is also a step ID in productLifecycle.
				call(s, 1).ID = "delete"
			}),
			want: nil,
		},
	})
}
