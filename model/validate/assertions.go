package validate

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// opsByTarget lists the operators applicable to each assertion target.
// Its keys are the supported targets.
var opsByTarget = map[model.AssertTarget][]model.AssertOp{
	model.TargetStatus: {
		model.OpEquals, model.OpNotEquals, model.OpIn,
		model.OpLt, model.OpLte, model.OpGt, model.OpGte,
	},
	model.TargetDuration: {
		model.OpLt, model.OpLte, model.OpGt, model.OpGte,
	},
	model.TargetHeader: {
		model.OpEquals, model.OpNotEquals, model.OpIn,
		model.OpExists, model.OpNotExists, model.OpContains, model.OpMatches,
	},
	model.TargetBody: {
		model.OpEquals, model.OpNotEquals, model.OpIn,
		model.OpExists, model.OpNotExists, model.OpContains, model.OpMatches,
		model.OpLt, model.OpLte, model.OpGt, model.OpGte, model.OpLength,
	},
}

// supportedTargets and supportedOps are listed in messages, in a stable order.
var (
	supportedTargets = []model.AssertTarget{
		model.TargetStatus, model.TargetBody, model.TargetHeader, model.TargetDuration,
	}
	supportedOps = []model.AssertOp{
		model.OpEquals, model.OpNotEquals, model.OpIn, model.OpExists, model.OpNotExists,
		model.OpContains, model.OpMatches, model.OpLt, model.OpLte, model.OpGt, model.OpGte, model.OpLength,
	}
)

// numericOps compare the actual value with a number.
var numericOps = []model.AssertOp{model.OpLt, model.OpLte, model.OpGt, model.OpGte, model.OpLength}

// validateAssertion checks one assertion of a request call: the target, the
// operator and whether they fit together, the path and the expected value.
//
// Values that are templates ("{{allowed}}") are not type-checked: their type
// is known only at run time. A nil value is a valid expectation for equals
// and notEquals (the actual value is null).
func (v *validator) validateAssertion(a model.Assertion, ptr diag.Pointer) {
	targetOK := v.validateAssertTarget(a.Target, ptr.Key("target"))
	opOK := v.validateAssertOp(a.Op, ptr.Key("op"))

	if targetOK && opOK && !slices.Contains(opsByTarget[a.Target], a.Op) {
		v.errorf(V0728, ptr.Key("op"), "operator %q is not applicable to %q, applicable: %q",
			a.Op, a.Target, opsByTarget[a.Target])
	}
	if targetOK {
		v.validateAssertPath(a.Target, a.Path, ptr.Key("path"))
	}
	if opOK {
		v.validateAssertValue(a.Op, a.Value, ptr.Key("value"))
	}
}

func (v *validator) validateAssertTarget(target model.AssertTarget, ptr diag.Pointer) bool {
	switch _, known := opsByTarget[target]; {
	case target == "":
		v.errorf(V0724, ptr, "assertion target is required")
	case !known:
		v.errorf(V0725, ptr, "unsupported assertion target %q, supported: %q", target, supportedTargets)
	default:
		return true
	}
	return false
}

func (v *validator) validateAssertOp(op model.AssertOp, ptr diag.Pointer) bool {
	switch {
	case op == "":
		v.errorf(V0726, ptr, "assertion operator is required")
	case !slices.Contains(supportedOps, op):
		v.errorf(V0727, ptr, "unsupported assertion operator %q, supported: %q", op, supportedOps)
	default:
		return true
	}
	return false
}

// validateAssertPath checks the path against the target: status and
// duration have no path, a header is selected by name, a body value by a
// JSONPath expression.
func (v *validator) validateAssertPath(target model.AssertTarget, path string, ptr diag.Pointer) {
	switch target {
	case model.TargetStatus, model.TargetDuration:
		if path != "" {
			v.errorf(V0729, ptr, "%q assertion has no path, got %q", target, path)
		}
	case model.TargetBody:
		switch {
		case path == "":
			v.errorf(V0730, ptr, "body assertion requires a JSONPath")
		case !strings.HasPrefix(path, "$"):
			v.errorf(V0731, ptr, "body assertion path %q must be a JSONPath starting with \"$\"", path)
		}
	case model.TargetHeader:
		if path == "" {
			v.errorf(V0730, ptr, "header assertion requires a header name")
		}
	}
}

// validateAssertValue checks the expected value against the operator.
func (v *validator) validateAssertValue(op model.AssertOp, val model.Value, ptr diag.Pointer) {
	switch op {
	case model.OpExists, model.OpNotExists:
		if val != nil {
			v.warnf(V0733, ptr, "operator %q ignores the value", op)
		}
		return
	case model.OpEquals, model.OpNotEquals:
		return // any value, including null
	}

	if val == nil {
		v.errorf(V0732, ptr, "operator %q requires a value", op)
		return
	}
	if isTemplate(val) {
		return
	}

	switch {
	case op == model.OpIn:
		if _, ok := val.([]any); !ok {
			v.errorf(V0734, ptr, "operator %q requires an array value, got %T", op, val)
		}
	case op == model.OpMatches:
		s, ok := val.(string)
		if !ok {
			v.errorf(V0735, ptr, "operator %q requires a regular expression string, got %T", op, val)
			break
		}
		if _, err := regexp.Compile(s); err != nil {
			v.errorf(V0735, ptr, "invalid regular expression %q: %v", s, err)
		}
	case slices.Contains(numericOps, op):
		if !isNumber(val) {
			v.errorf(V0736, ptr, "operator %q requires a number, got %T", op, val)
		}
	}
	// contains accepts any value: a substring or an array element.
}

// isNumber reports whether val is a number as produced by JSON decoding or
// written in Go code.
func isNumber(val model.Value) bool {
	switch n := val.(type) {
	case json.Number:
		_, err := n.Float64()
		return err == nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}
