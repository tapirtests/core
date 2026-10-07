package validate_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// Checks of request call steps: the reference, the strict input contract,
// the timeout and extracted variables; see scenario_steps.go.
// Step 0 of the probe scenario calls createProduct, whose inputs are
// auth.bearer, body.name and body.price (required) and body.categoryId.
func TestRequestCallSteps(t *testing.T) {
	const step0 = "/scenarios/probeScenario/steps/0"

	runBreakCases(t, []breakCase{
		// Request reference.
		{
			// Without a request the contract is not checked: the inputs of
			// the step would all be reported as unknown otherwise.
			name:   "missing request ID",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).RequestID = "" }),
			want:   []want{{validate.V0716, diag.Error, step0 + "/requestId"}},
		},
		{
			name:   "unknown request does not cascade to inputs",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).RequestID = "createProdukt" }),
			want:   []want{{validate.V0717, diag.Error, step0 + "/requestId"}},
		},
		{
			// The nil entry is reported once, with requests; steps calling it
			// are not checked against a contract that does not exist.
			name: "calls to a nil request are not checked",
			breaks: func(p *model.Project) {
				withScenario(func(s *model.ScenarioDef) {
					call(s, 0).Inputs["body.age"] = json.Number("1")
				})(p)
				p.Requests["createProduct"] = nil
			},
			want: []want{{validate.V0600, diag.Error, "/requests/createProduct"}},
		},

		// Strict contract: unknown inputs.
		{
			name:   "unknown input",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.age"] = json.Number("18") }),
			want:   []want{{validate.V0718, diag.Error, step0 + "/inputs/body.age"}},
		},
		{
			name:   "input without location",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["price"] = json.Number("1") }),
			want:   []want{{validate.V0718, diag.Error, step0 + "/inputs/price"}},
		},
		{
			// Names are matched exactly, even where the protocol is
			// case-insensitive: the validator points to the exact name.
			name: "header input name is matched exactly",
			breaks: func(p *model.Project) {
				withProbe(func(r *model.RequestDef) {})(p)
				withScenario(func(s *model.ScenarioDef) {
					s.Steps = []model.Step{&model.RequestCall{
						ID:        "probe",
						RequestID: "probe",
						Inputs: map[string]model.Value{
							"auth.bearer":    nil,
							"path.id":        json.Number("1"),
							"body.name":      "x",
							"header.x-trace": "abc",
						},
					}}
				})(p)
			},
			want: []want{{validate.V0718, diag.Error, step0 + "/inputs/header.x-trace"}},
		},

		// Strict contract: required inputs.
		{
			name:   "missing required input",
			breaks: withScenario(func(s *model.ScenarioDef) { delete(call(s, 0).Inputs, "body.price") }),
			want:   []want{{validate.V0719, diag.Error, step0 + "/inputs"}},
		},
		{
			name:   "missing required auth input",
			breaks: withScenario(func(s *model.ScenarioDef) { delete(call(s, 0).Inputs, "auth.bearer") }),
			want:   []want{{validate.V0719, diag.Error, step0 + "/inputs"}},
		},
		{
			name:   "explicit null counts as passed",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs["body.price"] = nil }),
			want:   nil,
		},
		{
			name:   "missing optional input is valid",
			breaks: withScenario(func(s *model.ScenarioDef) { delete(call(s, 0).Inputs, "body.categoryId") }),
			want:   nil,
		},
		{
			name:   "no inputs at all",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Inputs = nil }),
			want: []want{
				{validate.V0719, diag.Error, step0 + "/inputs"}, // auth.bearer
				{validate.V0719, diag.Error, step0 + "/inputs"}, // body.name
				{validate.V0719, diag.Error, step0 + "/inputs"}, // body.price
			},
		},
		{
			name: "unknown and missing inputs together",
			breaks: withScenario(func(s *model.ScenarioDef) {
				in := call(s, 0).Inputs
				delete(in, "body.price")
				in["body.cost"] = json.Number("10")
			}),
			want: []want{
				{validate.V0719, diag.Error, step0 + "/inputs"},
				{validate.V0718, diag.Error, step0 + "/inputs/body.cost"},
			},
		},

		// Timeout.
		{
			name:   "negative timeout",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Timeout = -time.Second }),
			want:   []want{{validate.V0720, diag.Error, step0 + "/timeout"}},
		},
		{
			name:   "zero timeout means default",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Timeout = 0 }),
			want:   nil,
		},

		// Extract.
		{
			name:   "empty extracted variable name",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract[""] = "$.body.id" }),
			want:   []want{{validate.V0721, diag.Error, step0 + "/extract/"}},
		},
		{
			name:   "extracted variable name with a dash",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["product-id"] = "$.body.id" }),
			want:   []want{{validate.V0722, diag.Error, step0 + "/extract/product-id"}},
		},
		{
			name:   "extracted variable name with a dot",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["product.id"] = "$.body.id" }),
			want:   []want{{validate.V0722, diag.Error, step0 + "/extract/product.id"}},
		},
		{
			name:   "extract path without $",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["productId"] = "body.id" }),
			want:   []want{{validate.V0723, diag.Error, step0 + "/extract/productId"}},
		},
		{
			name:   "empty extract path",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["productId"] = "" }),
			want:   []want{{validate.V0723, diag.Error, step0 + "/extract/productId"}},
		},
		{
			name:   "extract path with a wildcard",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["ids"] = "$.body.items[*].id" }),
			want:   []want{{validate.V0723, diag.Error, step0 + "/extract/ids"}},
		},
		{
			name: "extract path with a filter",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).Extract["cheap"] = "$.body.items[?(@.price < 10)]"
			}),
			want: []want{{validate.V0723, diag.Error, step0 + "/extract/cheap"}},
		},
		{
			name:   "malformed extract path",
			breaks: withScenario(func(s *model.ScenarioDef) { call(s, 0).Extract["productId"] = "$.body.items[" }),
			want:   []want{{validate.V0723, diag.Error, step0 + "/extract/productId"}},
		},
		{
			name: "exact extract paths of every form",
			breaks: withScenario(func(s *model.ScenarioDef) {
				ex := call(s, 0).Extract
				ex["status"] = "$.status"
				ex["requestId"] = "$.headers['X-Request-Id']"
				ex["firstTag"] = "$.body.tags[0]"
				ex["lastTag"] = "$.body.tags[-1]"
				ex["whole"] = "$.body"
			}),
			want: nil,
		},
		{
			name: "invalid name and path are both reported",
			breaks: withScenario(func(s *model.ScenarioDef) {
				call(s, 0).Extract["product-id"] = "body.id"
			}),
			want: []want{
				{validate.V0722, diag.Error, step0 + "/extract/product-id"},
				{validate.V0723, diag.Error, step0 + "/extract/product-id"},
			},
		},
	})
}
