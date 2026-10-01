package validate

import (
	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// validateScenarios checks every definition in the Scenario bucket: the
// signature (inputs and outputs) and the steps. Usages of scenarios (calls in
// groups) are checked with groups.
func (v *validator) validateScenarios() {
	base := diag.Root.Key("scenarios")
	for id, sc := range v.p.Scenarios {
		ptr := base.Key(string(id))
		if sc == nil {
			v.errorf(V0700, ptr, "scenario %q is empty", id)
			continue
		}

		switch sc.ID {
		case id:
		case "":
			v.errorf(V0701, ptr.Key("id"), "scenario ID is required")
		default:
			v.errorf(V0702, ptr.Key("id"), "scenario ID %q differs from its key %q", sc.ID, id)
		}

		v.validateScenarioInputs(sc.Inputs, ptr.Key("inputs"))
		v.validateScenarioOutputs(sc.Outputs, ptr.Key("outputs"))
		v.validateSteps(sc.Steps, ptr.Key("steps"))
	}
}

// validateScenarioInputs checks the inputs of a scenario signature.
func (v *validator) validateScenarioInputs(inputs []model.Param, ptr diag.Pointer) {
	seen := make(map[string]bool, len(inputs))
	for i, in := range inputs {
		inPtr := ptr.Index(i)
		if !v.validateDeclaredName(in.Name, seen, inPtr.Key("name"), V0703, V0704, V0705, "scenario input") {
			continue
		}
		if in.Required && in.Default != nil {
			v.warnf(V0710, inPtr.Key("default"),
				"input %q is required, so its default value is never used", in.Name)
		}
	}
}

// validateScenarioOutputs checks the outputs of a scenario signature.
func (v *validator) validateScenarioOutputs(outputs []model.Output, ptr diag.Pointer) {
	seen := make(map[string]bool, len(outputs))
	for i, out := range outputs {
		outPtr := ptr.Index(i)
		if !v.validateDeclaredName(out.Name, seen, outPtr.Key("name"), V0706, V0707, V0708, "scenario output") {
			continue
		}
		if out.Value == nil {
			v.errorf(V0709, outPtr.Key("value"), "output %q has no value", out.Name)
		}
	}
}

// validateDeclaredName checks a name declared in a list: it is not empty,
// it is an identifier, and it is not declared twice. Codes are given for
// these three problems in that order; what names the declared thing in
// messages. It returns false if the name is empty or a duplicate, i.e. when
// further checks of the element would only repeat the problem.
func (v *validator) validateDeclaredName(name string, seen map[string]bool, ptr diag.Pointer,
	empty, invalid, duplicate diag.Code, what string) bool {
	if name == "" {
		v.errorf(empty, ptr, "%s name is required", what)
		return false
	}
	if !identifierPattern.MatchString(name) {
		v.errorf(invalid, ptr, "%s name %q must match %s", what, name, identifierPattern)
	}
	if seen[name] {
		v.errorf(duplicate, ptr, "%s %q is declared more than once", what, name)
		return false
	}
	seen[name] = true
	return true
}

// validateSteps checks the steps of a scenario: IDs are unique within the
// scenario, and every step is checked according to its kind.
func (v *validator) validateSteps(steps []model.Step, ptr diag.Pointer) {
	if len(steps) == 0 {
		v.warnf(V0711, ptr, "scenario has no steps")
		return
	}

	seen := make(map[string]bool, len(steps))
	for i, step := range steps {
		stepPtr := ptr.Index(i)
		switch step := step.(type) {
		case nil:
			v.errorf(V0712, stepPtr, "step is empty")
		case *model.RequestCall:
			if step == nil {
				v.errorf(V0712, stepPtr, "step is empty")
				continue
			}
			v.validateStepID(step.ID, seen, stepPtr.Key("id"))
			v.validateRequestCall(step, stepPtr)
		default:
			v.errorf(V0715, stepPtr, "unsupported step kind %T", step)
		}
	}
}

// validateStepID checks that a step ID is set and unique within its scenario.
func (v *validator) validateStepID(id string, seen map[string]bool, ptr diag.Pointer) {
	switch {
	case id == "":
		v.errorf(V0713, ptr, "step ID is required")
	case seen[id]:
		v.errorf(V0714, ptr, "step ID %q is used more than once in the scenario", id)
	default:
		seen[id] = true
	}
}
