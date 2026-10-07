// Package extract takes values out of a response with JSONPath (RFC 9535).
//
// Only exact paths are supported for now: a path made of fields and indexes
// that points to one place and so yields one value, like $.body.items[0].name
// or $.headers['X-Request-Id']. Wildcards, filters, slices and descendant
// selectors can match many values; they are a planned extension and are
// rejected with a clear message until then.
package extract

import (
	"errors"
	"strings"

	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

// ErrNotExact is returned by Parse for a valid JSONPath that may select more
// than one value.
var ErrNotExact = errors.New(
	"wildcards, filters, slices and multiple selectors are not supported yet: the path must point to exactly one value")

// Path is a parsed exact JSONPath.
type Path struct {
	path *jsonpath.Path
}

// Parse parses an exact JSONPath. It returns an error for a malformed path
// and ErrNotExact for a path that is valid JSONPath but may select more than
// one value. Errors do not repeat the path: the caller adds its own context.
func Parse(path string) (*Path, error) {
	p, err := jsonpath.Parse(path)
	if err != nil {
		// Drop the library prefix: the caller adds its own context.
		return nil, errors.New(strings.TrimPrefix(err.Error(), "jsonpath: "))
	}
	if p.Query().Singular() == nil {
		return nil, ErrNotExact
	}
	return &Path{path: p}, nil
}

// Get returns the value the path points to in doc, a value built of
// map[string]any and []any as produced by JSON decoding. ok is false if doc
// has nothing at that place; a null found there is returned as (nil, true).
func (p *Path) Get(doc any) (value any, ok bool) {
	nodes := p.path.Select(doc)
	if len(nodes) == 0 {
		return nil, false
	}
	return nodes[0], true
}

// Step is one step of an exact path: a field of an object or an index of an
// array.
type Step struct {
	Field   string // object key; used when IsIndex is false
	Index   int    // array index, negative counts from the end; used when IsIndex is true
	IsIndex bool
}

// Steps returns the steps of the path from the root: "$.body.items[0]" is
// body, items, 0. The root path "$" has no steps.
func (p *Path) Steps() []Step {
	segments := p.path.Query().Segments()
	steps := make([]Step, 0, len(segments))
	for _, seg := range segments {
		// An exact path has exactly one selector per segment, a name or an
		// index; Parse rejects everything else.
		switch sel := seg.Selectors()[0].(type) {
		case spec.Name:
			steps = append(steps, Step{Field: string(sel)})
		case spec.Index:
			steps = append(steps, Step{Index: int(sel), IsIndex: true})
		}
	}
	return steps
}

// Part returns the part of the response document the path leads into, one of
// Parts, i.e. the first field of the path. It is empty for the root path "$"
// and for a path that does not start with a field.
func (p *Path) Part() string {
	steps := p.Steps()
	if len(steps) == 0 || steps[0].IsIndex {
		return ""
	}
	return steps[0].Field
}

// String returns the path in the normalized form of RFC 9535:
// $["body"]["items"][0].
func (p *Path) String() string {
	return p.path.String()
}
