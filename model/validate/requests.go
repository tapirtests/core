package validate

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// inAuth is the input location of credentials: auth.<scheme>. It is shared by
// all protocols; every other location is protocol-specific.
const inAuth = "auth"

// inputRules is the protocol-specific part of input contract validation.
// Each protocol implements it in its own file (requests_http.go, ...) and
// returns it from validateOperation; validateInputs applies the checks common
// to all protocols and delegates the rest.
type inputRules interface {
	// locations lists the input locations of the protocol, except "auth".
	locations() []string
	// allowsBare reports whether the location may be used without a field
	// name, e.g. "body" for the whole HTTP body.
	allowsBare(location string) bool
	// dedupKey returns the key under which duplicate inputs are detected,
	// e.g. lower-cased for case-insensitive HTTP headers.
	dedupKey(name, location string) string
	// checkInput checks one input with a known location. field is empty for
	// a bare location.
	checkInput(v *validator, in model.InputField, location, field string, ptr diag.Pointer)
	// finish runs after all inputs were checked; ptr points to the inputs.
	finish(v *validator, ptr diag.Pointer)
}

// validateRequests checks every definition in the Requests bucket: the
// operation and the input/output contract. Usages of requests (steps) are
// checked with scenarios.
func (v *validator) validateRequests() {
	base := diag.Root.Key("requests")
	for id, req := range v.p.Requests {
		ptr := base.Key(string(id))
		if req == nil {
			v.errorf(V0600, ptr, "request %q is empty", id)
			continue
		}

		switch req.ID {
		case id:
		case "":
			v.errorf(V0601, ptr.Key("id"), "request ID is required")
		default:
			v.errorf(V0602, ptr.Key("id"), "request ID %q differs from its key %q", req.ID, id)
		}

		rules := v.validateOperation(req.Operation, ptr.Key("operation"))
		v.validateInputs(req.Inputs, rules, ptr.Key("inputs"))
		v.validateResponses(req.Responses, ptr.Key("responses"))
	}
}

// validateOperation checks the protocol-specific part of a request and
// returns the input rules of its protocol, or nil if the protocol is unknown
// (no operation, unsupported protocol). Without rules only the checks common
// to all protocols are applied to the inputs.
func (v *validator) validateOperation(op model.Operation, ptr diag.Pointer) inputRules {
	switch op := op.(type) {
	case nil:
		v.errorf(V0603, ptr, "request operation is required")
		return nil
	case model.HTTPOperation:
		return v.validateHTTPOperation(op, ptr)
	case *model.HTTPOperation:
		if op == nil {
			v.errorf(V0603, ptr, "request operation is required")
			return nil
		}
		return v.validateHTTPOperation(*op, ptr)
	default:
		v.errorf(V0604, ptr, "unsupported operation protocol %q", op.Protocol())
		return nil
	}
}

// validateInputs checks the input contract of a request. The name of an
// input is "<location>.<field>"; the checks of names, duplicates and auth
// inputs are common to all protocols, the rest comes from rules.
func (v *validator) validateInputs(inputs []model.InputField, rules inputRules, ptr diag.Pointer) {
	seen := make(map[string]bool, len(inputs))
	for i, in := range inputs {
		inPtr := ptr.Index(i)
		v.validateSchema(in.Schema, inPtr.Key("schema"))

		namePtr := inPtr.Key("name")
		if in.Name == "" {
			v.errorf(V0611, namePtr, "input name is required")
			continue
		}

		location, field, hasField := strings.Cut(in.Name, ".")
		key := in.Name
		if rules != nil {
			key = rules.dedupKey(in.Name, location)
		}
		if seen[key] {
			v.errorf(V0614, namePtr, "input %q is declared more than once", in.Name)
			continue
		}
		seen[key] = true

		if location == inAuth {
			v.validateAuthInput(in.Name, field, namePtr)
			continue
		}
		if rules == nil {
			// Locations of an unknown protocol cannot be checked.
			continue
		}

		if !slices.Contains(rules.locations(), location) {
			v.errorf(V0612, namePtr, "input %q has unknown location %q, known: %q",
				in.Name, location, slices.Concat(rules.locations(), []string{inAuth}))
			continue
		}
		if !hasField && rules.allowsBare(location) {
			rules.checkInput(v, in, location, "", inPtr)
			continue
		}
		if field == "" {
			v.errorf(V0613, namePtr, "input %q has no field name after %q", in.Name, location+".")
			continue
		}
		rules.checkInput(v, in, location, field, inPtr)
	}

	if rules != nil {
		rules.finish(v, ptr)
	}
}

// validateAuthInput checks an auth.<scheme> input: the scheme must exist.
func (v *validator) validateAuthInput(name, scheme string, ptr diag.Pointer) {
	if scheme == "" {
		v.errorf(V0613, ptr, "input %q has no security scheme name after %q", name, inAuth+".")
		return
	}
	if _, ok := v.p.SecuritySchemes[scheme]; !ok {
		v.errorf(V0617, ptr, "input %q refers to unknown security scheme %q", name, scheme)
	}
}

// validateResponses checks the output contract: keys are status codes
// ("200") or "default". Protocols without status codes use only "default".
func (v *validator) validateResponses(responses map[string]model.ResponseDef, ptr diag.Pointer) {
	for key, resp := range responses {
		respPtr := ptr.Key(key)
		if !isResponseKey(key) {
			v.errorf(V0619, respPtr, "response key %q must be an HTTP status code (100-599) or \"default\"", key)
		}
		v.validateSchema(resp.Schema, respPtr.Key("schema"))
	}
}

func isResponseKey(key string) bool {
	if key == "default" {
		return true
	}
	code, err := strconv.Atoi(key)
	return err == nil && len(key) == 3 && code >= 100 && code <= 599
}

// validateSchema checks that a non-empty schema is valid JSON. The schema
// itself is interpreted by other packages; an empty one means "undocumented".
func (v *validator) validateSchema(s model.Schema, ptr diag.Pointer) {
	if len(s) > 0 && !json.Valid(s) {
		v.errorf(V0620, ptr, "schema is not valid JSON")
	}
}
