package validate

import (
	"strings"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/extract"
)

// responseParts lists the parts of the response document for messages:
// "$.status, $.headers, $.body, $.duration".
func responseParts() string {
	parts := make([]string, len(extract.Parts))
	for i, part := range extract.Parts {
		parts[i] = "$." + part
	}
	return strings.Join(parts, ", ")
}

// validateResponsePath checks a parsed path against the shape of the
// response document, which is known in advance:
//
//	$.status    a number
//	$.headers   an object of strings with lower-case names
//	$.body      whatever the API returns
//	$.duration  a number
//
// It is used for both extraction and assertions, so a path means the same
// thing in both. The root path "$" (the whole response) is accepted here;
// callers that cannot use it reject it themselves. It returns false if the
// path can never point to anything.
func (v *validator) validateResponsePath(p *extract.Path, path string, ptr diag.Pointer) bool {
	steps := p.Steps()
	if len(steps) == 0 {
		return true
	}

	switch part := p.Part(); part {
	case extract.PartBody:
		return true
	case extract.PartStatus, extract.PartDuration:
		if len(steps) > 1 {
			v.errorf(V0729, ptr, "path %q goes inside $.%s, which is a number", path, part)
			return false
		}
	case extract.PartHeaders:
		if len(steps) == 1 {
			return true // all headers, as an object
		}
		if len(steps) > 2 || steps[1].IsIndex {
			v.errorf(V0729, ptr,
				"path %q is not a header: a header is a string selected by name, like $.headers['content-type']", path)
			return false
		}
		if name := steps[1].Field; name != strings.ToLower(name) {
			v.warnf(V0737, ptr,
				"header name %q has upper-case letters: header names are lower-case in the response, use %q",
				name, strings.ToLower(name))
		}
	default:
		v.errorf(V0725, ptr, "path %q does not start with a part of the response: %s", path, responseParts())
		return false
	}
	return true
}
