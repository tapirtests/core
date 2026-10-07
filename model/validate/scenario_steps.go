package validate

import (
	"strings"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/extract"
	"github.com/tapirtests/core/model"
)

// validateRequestCall checks a step that calls a request: the reference, the
// strict input contract, the timeout, extracted variables and assertions.
func (v *validator) validateRequestCall(c *model.RequestCall, ptr diag.Pointer) {
	if req, ok := v.lookupRequest(c.RequestID, ptr.Key("requestId")); ok {
		v.validateCallInputs(c.Inputs, req, ptr.Key("inputs"))
	}
	// Templates are checked for every passed input, known to the request or
	// not: a syntax error is a problem of the step either way.
	for name, value := range c.Inputs {
		v.validateTemplates(value, ptr.Key("inputs").Key(name))
	}

	if c.Timeout < 0 {
		v.errorf(V0720, ptr.Key("timeout"), "timeout %s must not be negative", c.Timeout)
	}

	v.validateExtract(c.Extract, ptr.Key("extract"))

	for i, a := range c.Expect {
		v.validateAssertion(a, ptr.Key("expect").Index(i))
	}
}

// lookupRequest returns the request a step refers to. ok is false when the
// contract cannot be checked: the reference is missing or broken (reported
// here), or the request entry itself is nil (reported with requests).
func (v *validator) lookupRequest(id model.RequestID, ptr diag.Pointer) (*model.RequestDef, bool) {
	if id == "" {
		v.errorf(V0716, ptr, "request ID is required")
		return nil, false
	}
	req, found := v.p.Requests[id]
	if !found {
		v.errorf(V0717, ptr, "unknown request %q", id)
		return nil, false
	}
	return req, req != nil
}

// validateCallInputs checks the strict contract of a request call: it may
// pass only inputs the request declares, and must pass every required one.
// An explicit null counts as passed: it means "intentionally no value", e.g.
// a request without authorization.
func (v *validator) validateCallInputs(inputs map[string]model.Value, req *model.RequestDef, ptr diag.Pointer) {
	declared := make(map[string]bool, len(req.Inputs))
	for _, in := range req.Inputs {
		declared[in.Name] = true
	}

	for name := range inputs {
		if !declared[name] {
			v.errorf(V0718, ptr.Key(name), "request %q has no input %q", req.ID, name)
		}
	}

	for _, in := range req.Inputs {
		if !in.Required {
			continue
		}
		if _, passed := inputs[in.Name]; !passed {
			v.errorf(V0719, ptr, "required input %q of request %q is not passed (use null to send no value)",
				in.Name, req.ID)
		}
	}
}

// validateExtract checks extracted variables: names are identifiers declared
// in the scenario scope, paths are exact JSONPath expressions over the
// response. Whether a path exists in the response schema is not checked
// here.
func (v *validator) validateExtract(paths map[string]string, ptr diag.Pointer) {
	for name, path := range paths {
		varPtr := ptr.Key(name)
		switch {
		case name == "":
			v.errorf(V0721, varPtr, "extracted variable name is required")
		case !identifierPattern.MatchString(name):
			v.errorf(V0722, varPtr, "extracted variable name %q must match %s", name, identifierPattern)
		}
		v.validateNotReserved(name, varPtr, "extracted variable")
		if _, err := extract.Parse(path); err != nil {
			v.errorf(V0723, varPtr, "extract path %q: %v", path, err)
		}
	}
}

// isTemplate reports whether a value is a template string resolved at run
// time, like "{{allowedStatuses}}": its type cannot be checked statically.
func isTemplate(val model.Value) bool {
	s, ok := val.(string)
	return ok && strings.Contains(s, "{{")
}
