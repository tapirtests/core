package vars

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Caller produces values for function calls in templates: {{random.email}}.
// *random.Generator implements it; the indirection keeps this package
// independent of how values are generated.
type Caller interface {
	Call(name string, args []any) (any, error)
}

// Resolver evaluates templates against a scope.
//
// Scope is required. Env and Random may be nil: then templates that use
// {{env.*}} or {{random.*}} fail with an error instead of resolving.
type Resolver struct {
	Scope *Scope
	// Env returns the value of an env variable.
	Env func(name string) (value string, ok bool)
	// Random produces values for random.* calls.
	Random Caller
}

// Resolve returns value with all templates evaluated. It walks objects and
// arrays recursively and evaluates every string in them; object keys are not
// templates. The input is not modified: changed objects and arrays are
// rebuilt.
//
// A string that consists of exactly one expression, like "{{productId}}",
// resolves to the value of the expression with its type preserved: a number
// stays a number, an object stays an object. A string with an expression
// among other text, like "id-{{productId}}", resolves to a string.
//
// Values taken from the scope are returned as they are stored, without
// copying; the caller must not modify them.
//
// The error is a *ParseError for a malformed template and a *ResolveError
// for an expression that cannot be evaluated.
func (r Resolver) Resolve(value any) (any, error) {
	return r.resolve(value, "")
}

// resolve does the work of Resolve; at is the place of value inside the
// value being resolved, e.g. ".owner.tags[1]".
func (r Resolver) resolve(value any, at string) (any, error) {
	switch v := value.(type) {
	case string:
		out, err := r.resolveString(v)
		if err != nil {
			return nil, withPlace(err, at)
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			resolved, err := r.resolve(item, at+Segment{Field: key}.String())
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			resolved, err := r.resolve(item, at+Segment{Index: i, IsIndex: true}.String())
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	default:
		return value, nil
	}
}

func (r Resolver) resolveString(s string) (any, error) {
	t, err := Parse(s)
	if err != nil {
		return nil, err
	}
	return r.ResolveTemplate(t)
}

// ResolveTemplate evaluates a parsed template; see Resolve for the rules.
func (r Resolver) ResolveTemplate(t *Template) (any, error) {
	if t.IsStatic() {
		// Not t.Source(): escaped braces are already unescaped in the parts.
		return textOf(t), nil
	}

	if expr, ok := t.Single(); ok {
		return r.eval(t, expr)
	}

	var b strings.Builder
	for _, part := range t.Parts() {
		if part.Expr == nil {
			b.WriteString(part.Text)
			continue
		}
		value, err := r.eval(t, part.Expr)
		if err != nil {
			return nil, err
		}
		b.WriteString(stringify(value))
	}
	return b.String(), nil
}

// textOf joins the text parts of a static template.
func textOf(t *Template) string {
	parts := t.Parts()
	if len(parts) == 1 {
		return parts[0].Text
	}
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(part.Text)
	}
	return b.String()
}

// eval evaluates one expression of t.
func (r Resolver) eval(t *Template, e *Expr) (any, error) {
	fail := func(format string, args ...any) error {
		return &ResolveError{Src: t.Source(), Expr: e.String(), Pos: e.Pos, Msg: fmt.Sprintf(format, args...)}
	}

	switch e.Kind {
	case ExprEnv:
		if r.Env == nil {
			return nil, fail("env variables are not available here")
		}
		value, ok := r.Env(e.Name)
		if !ok {
			return nil, fail("unknown env variable %q", e.Name)
		}
		return value, nil

	case ExprCall:
		if r.Random == nil {
			return nil, fail("random values are not available here")
		}
		value, err := r.Random.Call(e.Name, e.Args)
		if err != nil {
			return nil, fail("%v", err)
		}
		return value, nil

	default:
		if r.Scope == nil {
			return nil, fail("variables are not available here")
		}
		value, ok := r.Scope.Get(e.Name)
		if !ok {
			return nil, fail("unknown variable %q", e.Name)
		}
		return walk(value, e, fail)
	}
}

// walk follows the path of a variable expression into its value: every
// segment takes a field of an object or an element of an array.
func walk(value any, e *Expr, fail func(format string, args ...any) error) (any, error) {
	// where is the part of the expression walked so far, for messages:
	// "world", then "world[0]", then "world[0].title".
	where := e.Name
	for _, seg := range e.Path {
		if seg.IsIndex {
			arr, ok := value.([]any)
			if !ok {
				return nil, fail("%s is %s, not an array: cannot take element %d", where, kindOf(value), seg.Index)
			}
			if seg.Index >= len(arr) {
				return nil, fail("%s has %d element(s): index %d is out of range", where, len(arr), seg.Index)
			}
			value = arr[seg.Index]
		} else {
			obj, ok := value.(map[string]any)
			if !ok {
				return nil, fail("%s is %s, not an object: cannot take field %q", where, kindOf(value), seg.Field)
			}
			field, ok := obj[seg.Field]
			if !ok {
				return nil, fail("%s has no field %q", where, seg.Field)
			}
			value = field
		}
		where += seg.String()
	}
	return value, nil
}

// stringify converts a value to the text inserted into a string template.
// Strings are inserted as they are, numbers and booleans as written, null as
// "null", objects and arrays as compact JSON.
func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	}
}

// kindOf names the JSON kind a value with an article, for messages.
func kindOf(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "a string"
	case bool:
		return "a boolean"
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	case json.Number, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "a number"
	default:
		return fmt.Sprintf("a %T", value)
	}
}

// ResolveError is an expression that could not be evaluated: an unknown
// variable, a path that does not exist in the value, a failed call.
type ResolveError struct {
	Src  string // the template source
	Expr string // the expression in canonical form, without braces
	Pos  int    // byte offset of the expression in Src
	// At is the place of the template inside the resolved value, e.g.
	// ".owner.tags[1]"; empty if the value itself is the template.
	At  string
	Msg string
}

func (e *ResolveError) Error() string {
	place := ""
	if e.At != "" {
		place = " (at " + e.At + ")"
	}
	return fmt.Sprintf("template %q%s: %s", e.Src, place, e.Msg)
}

// withPlace records where inside a resolved value the failing template is.
func withPlace(err error, at string) error {
	if rerr, ok := err.(*ResolveError); ok && at != "" {
		rerr.At = at
	}
	return err
}
