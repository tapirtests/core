package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// HTTP input locations: the prefix of an input name tells which part of the
// HTTP request the value goes to.
const (
	inPath     = "path"
	inQuery    = "query"
	inHeader   = "header"
	inCookie   = "cookie"
	inBody     = "body"
	inFormData = "formData"
)

var httpInputLocations = []string{inPath, inQuery, inHeader, inCookie, inBody, inFormData}

// supportedHTTPMethods lists the methods an HTTPOperation may use. Methods are
// upper case; importers normalize the lower-case keys of Swagger.
var supportedHTTPMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// validateHTTPOperation checks the method and the path template and returns
// the input rules of the operation.
func (v *validator) validateHTTPOperation(op model.HTTPOperation, ptr diag.Pointer) *httpRules {
	switch {
	case op.Method == "":
		v.errorf(V0605, ptr.Key("method"), "HTTP method is required")
	case !slices.Contains(supportedHTTPMethods, op.Method):
		v.errorf(V0606, ptr.Key("method"), "unsupported HTTP method %q, supported: %q",
			op.Method, supportedHTTPMethods)
	}

	rules := &httpRules{pathInputs: make(map[string]bool)}

	pathPtr := ptr.Key("path")
	switch {
	case op.Path == "":
		v.errorf(V0607, pathPtr, "HTTP path is required")
		return rules
	case !strings.HasPrefix(op.Path, "/"):
		v.errorf(V0608, pathPtr, "HTTP path %q must start with \"/\"", op.Path)
	}

	params, err := parsePathTemplate(op.Path)
	if err != nil {
		v.errorf(V0609, pathPtr, "HTTP path %q: %v", op.Path, err)
		return rules
	}
	rules.params, rules.paramsKnown = params, true
	return rules
}

// parsePathTemplate returns the parameter names of a path template like
// "/shops/{shopId}/products/{id}".
func parsePathTemplate(path string) ([]string, error) {
	var names []string
	for rest := path; rest != ""; {
		open := strings.IndexAny(rest, "{}")
		if open < 0 {
			break
		}
		if rest[open] == '}' {
			return nil, fmt.Errorf(`unexpected "}"`)
		}
		closeIdx := strings.IndexAny(rest[open+1:], "{}")
		if closeIdx < 0 || rest[open+1+closeIdx] == '{' {
			return nil, fmt.Errorf(`unclosed "{"`)
		}
		name := rest[open+1 : open+1+closeIdx]
		if name == "" {
			return nil, fmt.Errorf(`empty parameter name "{}"`)
		}
		names = append(names, name)
		rest = rest[open+1+closeIdx+1:]
	}
	return names, nil
}

// httpRules are the input rules of an HTTP operation. They also collect what
// is needed for the checks made after all inputs were seen.
type httpRules struct {
	// params are the {names} of the path template. paramsKnown is false when
	// the template is missing or malformed: then path inputs are not
	// cross-checked, so one broken template does not cause a cascade.
	params      []string
	paramsKnown bool

	pathInputs   map[string]bool // fields of the path.<field> inputs
	wholeBodyPtr diag.Pointer    // pointer to the "body" input, if any
	hasWholeBody bool
	hasBodyField bool // whether any body.<field> input exists
}

func (r *httpRules) locations() []string { return httpInputLocations }

// allowsBare permits "body" alone: the whole request body, for bodies that
// are not objects (arrays, strings).
func (r *httpRules) allowsBare(location string) bool { return location == inBody }

// dedupKey makes header inputs case-insensitive: X-Trace and x-trace are the
// same HTTP header. Other locations are case-sensitive.
func (r *httpRules) dedupKey(name, location string) string {
	if location == inHeader {
		return strings.ToLower(name)
	}
	return name
}

func (r *httpRules) checkInput(v *validator, in model.InputField, location, field string, ptr diag.Pointer) {
	switch location {
	case inBody:
		if field == "" {
			r.hasWholeBody, r.wholeBodyPtr = true, ptr.Key("name")
		} else {
			r.hasBodyField = true
		}
	case inPath:
		r.pathInputs[field] = true
		if r.paramsKnown && !slices.Contains(r.params, field) {
			v.errorf(V0615, ptr.Key("name"), "input %q has no {%s} in the path template", in.Name, field)
		}
		if !in.Required {
			v.errorf(V0616, ptr.Key("required"), "path input %q must be required", in.Name)
		}
	}
}

func (r *httpRules) finish(v *validator, ptr diag.Pointer) {
	if r.hasWholeBody && r.hasBodyField {
		v.errorf(V0618, r.wholeBodyPtr, `whole-body input "body" cannot be combined with body.<field> inputs`)
	}
	if r.paramsKnown {
		for _, name := range r.params {
			if !r.pathInputs[name] {
				v.errorf(V0610, ptr, "path parameter {%s} has no %q input", name, inPath+"."+name)
			}
		}
	}
}
