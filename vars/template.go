package vars

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Namespaces with a special meaning as the first name of an expression.
// They cannot be used as variable names.
const (
	nsEnv    = "env"
	nsRandom = "random"
)

// Template is a parsed string that may contain "{{...}}" expressions.
// A template is immutable and safe for concurrent use.
type Template struct {
	src   string
	parts []Part
}

// Part is a piece of a template: either literal text or an expression.
type Part struct {
	Text string // literal text; used when Expr is nil
	Expr *Expr  // expression, or nil for a text part
}

// ExprKind tells what an expression refers to.
type ExprKind int

const (
	// ExprVar is a variable with an optional path: product.tags[0].
	ExprVar ExprKind = iota + 1
	// ExprEnv is an env variable: env.BASE_URL.
	ExprEnv
	// ExprCall is a call of a random data function: random.int(1, 999).
	ExprCall
)

// Expr is one "{{...}}" expression of a template.
type Expr struct {
	Kind ExprKind
	// Pos is the byte offset of the opening "{{" in the template source.
	Pos int

	// Name is the variable name (ExprVar), the env variable name (ExprEnv)
	// or the function name without its namespace (ExprCall): "int".
	Name string
	// Path is the path into the variable value; ExprVar only.
	Path []Segment
	// Args are the literal arguments of the call, json.Number or string;
	// ExprCall only.
	Args []any
}

// Segment is one step of a path into a value: a field of an object or an
// index of an array.
type Segment struct {
	Field   string // object key; used when IsIndex is false
	Index   int    // array index; used when IsIndex is true
	IsIndex bool
}

// Source returns the string the template was parsed from.
func (t *Template) Source() string { return t.src }

// Parts returns the pieces of the template in order. The slice must not be
// modified.
func (t *Template) Parts() []Part { return t.parts }

// IsStatic reports whether the template has no expressions, i.e. it is plain
// text.
func (t *Template) IsStatic() bool {
	for _, p := range t.parts {
		if p.Expr != nil {
			return false
		}
	}
	return true
}

// Single returns the expression of a template that consists of exactly one
// expression and nothing else, like "{{productId}}". Such a template resolves
// to the value of the expression with its type preserved, not to a string.
func (t *Template) Single() (*Expr, bool) {
	if len(t.parts) == 1 && t.parts[0].Expr != nil {
		return t.parts[0].Expr, true
	}
	return nil, false
}

// Exprs returns the expressions of the template in order.
func (t *Template) Exprs() []*Expr {
	var exprs []*Expr
	for _, p := range t.parts {
		if p.Expr != nil {
			exprs = append(exprs, p.Expr)
		}
	}
	return exprs
}

// String returns the template in its canonical form: expressions without
// extra spaces, literal braces escaped. Parsing the result gives an equal
// template.
func (t *Template) String() string {
	var b strings.Builder
	for _, p := range t.parts {
		if p.Expr != nil {
			b.WriteString("{{")
			b.WriteString(p.Expr.String())
			b.WriteString("}}")
			continue
		}
		b.WriteString(strings.ReplaceAll(p.Text, "{{", `\{{`))
	}
	return b.String()
}

// String returns the expression in its canonical form, without braces:
// product.tags[0], env.BASE_URL, random.int(1, 999).
func (e *Expr) String() string {
	var b strings.Builder
	switch e.Kind {
	case ExprEnv:
		b.WriteString(nsEnv + "." + e.Name)
	case ExprCall:
		b.WriteString(nsRandom + "." + e.Name)
		if len(e.Args) > 0 {
			b.WriteByte('(')
			for i, arg := range e.Args {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(literalString(arg))
			}
			b.WriteByte(')')
		}
	default:
		b.WriteString(e.Name)
		for _, s := range e.Path {
			b.WriteString(s.String())
		}
	}
	return b.String()
}

// String returns the segment as written in a path: ".field", "[0]" or
// `["any key"]` for keys that are not identifiers.
func (s Segment) String() string {
	switch {
	case s.IsIndex:
		return "[" + strconv.Itoa(s.Index) + "]"
	case isIdent(s.Field):
		return "." + s.Field
	default:
		return "[" + quote(s.Field) + "]"
	}
}

func literalString(v any) string {
	switch v := v.(type) {
	case json.Number:
		return v.String()
	case string:
		return quote(v)
	default:
		return fmt.Sprint(v)
	}
}

// quote writes s as a string literal of the template language: in double
// quotes, with backslashes and double quotes escaped.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// ParseError is a syntax error in a template.
type ParseError struct {
	Src string // the template source
	Pos int    // byte offset of the error in Src
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("template %q: %s at offset %d", e.Src, e.Msg, e.Pos)
}
