package validate_test

import (
	"encoding/json"
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// Checks of templates, wherever a template may be written, and of names
// reserved by the template language; see templates.go.
func TestTemplates(t *testing.T) {
	const (
		step0    = "/scenarios/probeScenario/steps/0"
		products = "/root/main/groups/0"
	)

	runBreakCases(t, []breakCase{
		// What is checked: syntax, random calls, env references.
		{
			name:   "syntax error",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.name"] = "{{name" }),
			want:   []want{{validate.V0900, diag.Error, step0 + "/inputs/body.name"}},
		},
		{
			name:   "unknown random function",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.name"] = "{{random.emial}}" }),
			want:   []want{{validate.V0901, diag.Error, step0 + "/inputs/body.name"}},
		},
		{
			name:   "invalid random arguments",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.price"] = "{{random.int(10, 1)}}" }),
			want:   []want{{validate.V0901, diag.Error, step0 + "/inputs/body.price"}},
		},
		{
			name:   "undeclared env variable",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.name"] = "{{env.NOPE}}" }),
			want:   []want{{validate.V0902, diag.Error, step0 + "/inputs/body.name"}},
		},
		{
			name: "valid templates of every kind",
			breaks: withScenario(func(s *model.ScenarioDef) {
				in := call(s, 0).Inputs
				in["body.name"] = "{{env.SELLER_USER}}-{{random.string(6)}}-{{name}}"
				in["body.price"] = "{{random.float(1, 99.5)}}"
			}),
			want: nil,
		},
		{
			// Whether a variable exists depends on what runs before the
			// step; that is not checked here.
			name:   "unknown variable is not reported",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.name"] = "{{whoKnows.field[0]}}" }),
			want:   nil,
		},
		{
			name:   "escaped braces are not a template",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.name"] = `\{{env.NOPE}}` }),
			want:   nil,
		},
		{
			name: "several problems in one string",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).Inputs["body.name"] = "{{env.NOPE}} {{random.nope}} {{env.NOPE2}}"
			}),
			want: []want{
				{validate.V0902, diag.Error, step0 + "/inputs/body.name"},
				{validate.V0901, diag.Error, step0 + "/inputs/body.name"},
				{validate.V0902, diag.Error, step0 + "/inputs/body.name"},
			},
		},

		// Templates deep inside a value: the pointer leads to the string.
		{
			name: "template inside an object and an array",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).Inputs["body.name"] = map[string]any{
					"first": "ok {{name}}",
					"tags":  []any{"a", "{{broken"},
				}
			}),
			want: []want{{validate.V0900, diag.Error, step0 + "/inputs/body.name/tags/1"}},
		},

		// Where templates are checked.
		{
			name:   "base URL",
			breaks: func(p *model.Project) { p.BaseURL = "{{env.API_URL}}/v1" },
			want:   []want{{validate.V0902, diag.Error, "/baseUrl"}},
		},
		{
			name:   "scenario input default",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Default = "{{random.string()}}" }),
			want:   []want{{validate.V0901, diag.Error, "/scenarios/probeScenario/inputs/1/default"}},
		},
		{
			name:   "scenario output value",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Outputs[0].Value = "{{productId" }),
			want:   []want{{validate.V0900, diag.Error, "/scenarios/probeScenario/outputs/0/value"}},
		},
		{
			// Templates are checked even when the contract cannot be.
			name: "input of a step with an unknown request",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).RequestID = "nope"
				call(s, 0).Inputs["body.name"] = "{{"
			}),
			want: []want{
				{validate.V0717, diag.Error, step0 + "/requestId"},
				{validate.V0900, diag.Error, step0 + "/inputs/body.name"},
			},
		},
		{
			name: "assertion value",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).Expect[1].Value = "{{env.NOPE}}"
			}),
			want: []want{{validate.V0902, diag.Error, step0 + "/expect/1/value"}},
		},
		{
			name: "assertion value inside an array",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 1).Expect[0].Value = []any{json.Number("401"), "{{random.nope}}"}
			}),
			want: []want{{validate.V0901, diag.Error, "/scenarios/probeScenario/steps/1/expect/0/value/1"}},
		},
		{
			name:   "group variable",
			breaks: func(p *model.Project) { p.Root.Vars = map[string]model.Value{"currency": "{{env.CURRENCY}}"} },
			want:   []want{{validate.V0902, diag.Error, "/root/vars/currency"}},
		},
		{
			name:   "scenario call input",
			breaks: func(p *model.Project) { p.Root.Setup[0].Inputs["username"] = "{{env.NOPE}}" },
			want:   []want{{validate.V0902, diag.Error, "/root/setup/0/inputs/username"}},
		},
		{
			name: "scenario call input of an unknown scenario",
			breaks: func(p *model.Project) {
				p.Root.Setup[0].ScenarioID = "nope"
				p.Root.Setup[0].Inputs["username"] = "{{"
			},
			want: []want{
				{validate.V0809, diag.Error, "/root/setup/0/scenarioId"},
				{validate.V0900, diag.Error, "/root/setup/0/inputs/username"},
			},
		},
		{
			// A JSONPath is not a template.
			name:   "extract path is not a template",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["x"] = "$.body['{{']" }),
			want:   nil,
		},

		// Renaming a used env variable is seen from both sides.
		{
			name:   "renamed env variable breaks its users",
			breaks: func(p *model.Project) { p.Env[1].Name = "SELLER_LOGIN" },
			want:   []want{{validate.V0902, diag.Error, "/root/setup/0/inputs/username"}},
		},

		// Reserved names.
		{
			name:   "scenario input named env",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Name = "env" }),
			want:   []want{{validate.V0903, diag.Error, "/scenarios/probeScenario/inputs/1/name"}},
		},
		{
			name:   "scenario input named random",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Name = "random" }),
			want:   []want{{validate.V0903, diag.Error, "/scenarios/probeScenario/inputs/1/name"}},
		},
		{
			name:   "extracted variable named env",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["env"] = "$.body.env" }),
			want:   []want{{validate.V0903, diag.Error, step0 + "/extract/env"}},
		},
		{
			name:   "group variable named random",
			breaks: func(p *model.Project) { p.Root.Vars = map[string]model.Value{"random": json.Number("1")} },
			want:   []want{{validate.V0903, diag.Error, "/root/vars/random"}},
		},
		{
			name:   "alias named env",
			breaks: func(p *model.Project) { p.Root.Setup[0].Alias = "env" },
			want:   []want{{validate.V0903, diag.Error, "/root/setup/0/alias"}},
		},
		{
			// The implicit alias is the scenario ID.
			name: "scenario named random called without an alias",
			breaks: func(p *model.Project) {
				p.Scenarios["random"] = &model.ScenarioDef{ID: "random", Steps: p.Scenarios["createProductUnauthorized"].Steps}
				p.Root.Setup = append(p.Root.Setup, model.ScenarioCall{ScenarioID: "random"})
			},
			want: []want{{validate.V0903, diag.Error, "/root/setup/2/alias"}},
		},
		{
			name:   "output renamed to env",
			breaks: func(p *model.Project) { p.Root.Setup[0].Outputs["token"] = "env" },
			want:   []want{{validate.V0903, diag.Error, "/root/setup/0/outputs/token"}},
		},
		{
			// An output is read as {{alias.env}}, which is fine.
			name:   "scenario output may be named env",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Outputs[0].Name = "env" }),
			want:   nil,
		},
		{
			name:   "names that only start like a reserved one",
			breaks: withScenario(func(s *model.ScenarioDef) { s.Inputs[1].Name = "environment" }),
			want:   nil,
		},

		// A plain variable must not share a name with a call alias of the
		// group: outputs are stored under the alias.
		{
			name:   "renamed output clashes with an alias",
			breaks: func(p *model.Project) { p.Root.Setup[0].Outputs["token"] = "userLogin" },
			want:   []want{{validate.V0815, diag.Error, "/root/setup/0/outputs/token"}},
		},
		{
			name: "group variable clashes with an implicit alias",
			breaks: func(p *model.Project) {
				p.Root.Main.Groups[0].Vars = map[string]model.Value{"productLifecycle": "x"}
			},
			want: []want{{validate.V0815, diag.Error, products + "/vars/productLifecycle"}},
		},
		{
			name: "clash with an alias declared in a later section",
			breaks: func(p *model.Project) {
				g := p.Root.Main.Groups[0]
				g.Setup = []model.ScenarioCall{{
					ScenarioID: "login",
					Inputs:     map[string]model.Value{"username": "u", "password": "p"},
					Outputs:    map[string]string{"token": "cleanupProduct"},
				}}
			},
			want: []want{{validate.V0815, diag.Error, products + "/setup/0/outputs/token"}},
		},
		{
			// Scopes are separate: an alias of the parent group is just
			// shadowed, like any variable.
			name: "variable named like an alias of another group",
			breaks: func(p *model.Project) {
				p.Root.Main.Groups[0].Vars = map[string]model.Value{"sellerLogin": "x"}
			},
			want: nil,
		},
	})
}
