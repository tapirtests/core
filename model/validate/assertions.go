package validate

import (
	"encoding/json"
	"regexp"
	"slices"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/extract"
	"github.com/tapirtests/core/model"
)

// Operators applicable to the parts of the response whose type is known in
// advance. A value inside the body may be of any type, so every operator is
// applicable to it.
var (
	// statusOps apply to $.status, a number.
	statusOps = []model.AssertOp{
		model.OpEquals, model.OpNotEquals, model.OpIn,
		model.OpLt, model.OpLte, model.OpGt, model.OpGte,
	}
	// durationOps apply to $.duration, a number that is never equal to
	// anything predictable.
	durationOps = []model.AssertOp{model.OpLt, model.OpLte, model.OpGt, model.OpGte}
	// headerOps apply to a header, a string that may be absent.
	headerOps = []model.AssertOp{
		model.OpEquals, model.OpNotEquals, model.OpIn,
		model.OpExists, model.OpNotExists, model.OpContains, model.OpMatches,
	}
)

// supportedOps are listed in messages, in a stable order.
var supportedOps = []model.AssertOp{
	model.OpEquals, model.OpNotEquals, model.OpIn, model.OpExists, model.OpNotExists,
	model.OpContains, model.OpMatches, model.OpLt, model.OpLte, model.OpGt, model.OpGte, model.OpLength,
}

// numericOps compare the actual value with a number.
var numericOps = []model.AssertOp{model.OpLt, model.OpLte, model.OpGt, model.OpGte, model.OpLength}

// validateAssertion checks one assertion of a request call: the path, the
// operator, whether the operator fits what the path points to, and the
// expected value.
//
// Values that are templates ("{{allowed}}") are not type-checked: their type
// is known only at run time. A nil value is a valid expectation for equals
// and notEquals (the actual value is null).
func (v *validator) validateAssertion(a model.Assertion, ptr diag.Pointer) {
	path := v.validateAssertPath(a.Path, ptr.Key("path"))
	opOK := v.validateAssertOp(a.Op, ptr.Key("op"))

	if path != nil && opOK {
		if what, ops := applicableOps(path); ops != nil && !slices.Contains(ops, a.Op) {
			v.errorf(V0728, ptr.Key("op"), "operator %q is not applicable to %s, applicable: %q", a.Op, what, ops)
		}
	}
	if opOK {
		v.validateAssertValue(a.Op, a.Value, ptr.Key("value"))
	}
	v.validateTemplates(a.Value, ptr.Key("value"))
}

// validateAssertPath checks the path of an assertion and returns it parsed,
// or nil if it is unusable. An assertion checks something inside the
// response, so unlike extraction it cannot point to the response as a whole.
func (v *validator) validateAssertPath(path string, ptr diag.Pointer) *extract.Path {
	if path == "" {
		v.errorf(V0730, ptr, "assertion path is required, e.g. $.status or $.body.id")
		return nil
	}
	p, err := extract.Parse(path)
	if err != nil {
		v.errorf(V0731, ptr, "assertion path %q: %v", path, err)
		return nil
	}
	if len(p.Steps()) == 0 {
		v.errorf(V0725, ptr, "assertion path %q points to the whole response; it must start with one of %s",
			path, responseParts())
		return nil
	}
	if !v.validateResponsePath(p, path, ptr) {
		return nil
	}
	return p
}

// applicableOps returns the operators applicable to what the path points to
// and a name of that thing for messages. ops is nil when any operator is
// applicable.
func applicableOps(p *extract.Path) (what string, ops []model.AssertOp) {
	steps := p.Steps()
	switch p.Part() {
	case extract.PartStatus:
		return "the status", statusOps
	case extract.PartDuration:
		return "the duration", durationOps
	case extract.PartHeaders:
		if len(steps) == 2 {
			return "a header", headerOps
		}
	}
	return "", nil
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
