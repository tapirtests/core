package vars

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	openBraces  = "{{"
	closeBraces = "}}"
	escapedOpen = `\{{`
)

// Parse parses a string that may contain "{{...}}" expressions. A string
// without expressions is a valid template of a single text part. On a syntax
// error it returns a *ParseError.
func Parse(src string) (*Template, error) {
	t := &Template{src: src}

	// Fast path: most strings in a project are plain text.
	if !strings.Contains(src, openBraces) {
		if src != "" {
			t.parts = []Part{{Text: src}}
		}
		return t, nil
	}

	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			t.parts = append(t.parts, Part{Text: text.String()})
			text.Reset()
		}
	}

	for i := 0; i < len(src); {
		switch {
		case strings.HasPrefix(src[i:], escapedOpen):
			text.WriteString(openBraces)
			i += len(escapedOpen)
		case strings.HasPrefix(src[i:], openBraces):
			flush()
			p := parser{src: src, start: i, pos: i + len(openBraces)}
			expr, err := p.expr()
			if err != nil {
				return nil, err
			}
			t.parts = append(t.parts, Part{Expr: expr})
			i = p.pos
		default:
			text.WriteByte(src[i])
			i++
		}
	}
	flush()
	return t, nil
}

// parser parses one expression. pos is just after the opening braces when it
// starts and just after the closing braces when it succeeds.
type parser struct {
	src   string
	start int // offset of the opening "{{"
	pos   int
}

// errorf creates a ParseError at the current position. Running out of input
// inside an expression always means the braces were not closed, whatever the
// parser expected next, so that is what gets reported.
func (p *parser) errorf(format string, args ...any) error {
	if p.pos >= len(p.src) {
		return &ParseError{Src: p.src, Pos: p.start, Msg: `unclosed "{{"`}
	}
	return &ParseError{Src: p.src, Pos: p.pos, Msg: fmt.Sprintf(format, args...)}
}

// expr parses the expression and the closing braces.
func (p *parser) expr() (*Expr, error) {
	p.skipSpace()
	if p.hasPrefix(closeBraces) {
		return nil, p.errorf("empty expression")
	}

	name, err := p.ident("a name")
	if err != nil {
		return nil, err
	}

	e := &Expr{Pos: p.start, Name: name}
	switch name {
	case nsEnv:
		err = p.env(e)
	case nsRandom:
		err = p.call(e)
	default:
		err = p.variable(e)
	}
	if err != nil {
		return nil, err
	}

	p.skipSpace()
	if !p.hasPrefix(closeBraces) {
		return nil, p.errorf(`unexpected %s, expected "}}"`, p.describeNext())
	}
	p.pos += len(closeBraces)
	return e, nil
}

// env parses the rest of "env.NAME". Env values are strings, so no path may
// follow the name.
func (p *parser) env(e *Expr) error {
	if !p.consume('.') {
		return p.errorf(`expected ".NAME" after "env"`)
	}
	name, err := p.ident("an env variable name")
	if err != nil {
		return err
	}
	e.Kind, e.Name = ExprEnv, name

	if c := p.peek(); c == '.' || c == '[' {
		return p.errorf("env variable %q is a string and has no fields or elements", name)
	}
	return nil
}

// call parses the rest of "random.fn" or "random.fn(args)".
func (p *parser) call(e *Expr) error {
	if !p.consume('.') {
		return p.errorf(`expected ".function" after "random"`)
	}
	name, err := p.ident("a function name")
	if err != nil {
		return err
	}
	e.Kind, e.Name = ExprCall, name

	if p.consume('(') {
		if e.Args, err = p.args(); err != nil {
			return err
		}
	}
	if c := p.peek(); c == '.' || c == '[' {
		return p.errorf("the result of random.%s cannot be followed by a path", name)
	}
	return nil
}

// args parses call arguments up to and including the closing parenthesis.
// Arguments are literals: numbers and strings.
func (p *parser) args() ([]any, error) {
	var args []any
	p.skipSpace()
	if p.consume(')') {
		return args, nil
	}
	for {
		p.skipSpace()
		arg, err := p.literal()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		p.skipSpace()
		switch {
		case p.consume(')'):
			return args, nil
		case p.consume(','):
		default:
			return nil, p.errorf(`unexpected %s, expected "," or ")"`, p.describeNext())
		}
	}
}

// literal parses a number or a string literal.
func (p *parser) literal() (any, error) {
	switch c := p.peek(); {
	case c == '"' || c == '\'':
		return p.str()
	case c == '-' || isDigit(c):
		return p.number()
	default:
		return nil, p.errorf("unexpected %s, expected a number or a string", p.describeNext())
	}
}

// variable parses the path that follows a variable name.
func (p *parser) variable(e *Expr) error {
	e.Kind = ExprVar
	for {
		switch {
		case p.consume('.'):
			field, err := p.ident("a field name")
			if err != nil {
				return err
			}
			e.Path = append(e.Path, Segment{Field: field})
		case p.consume('['):
			seg, err := p.subscript()
			if err != nil {
				return err
			}
			e.Path = append(e.Path, seg)
		case p.peek() == '(':
			return p.errorf("%q is not a function: only random.* functions can be called", e.String())
		default:
			return nil
		}
	}
}

// subscript parses the inside of "[...]" and the closing bracket: an array
// index or a quoted object key.
func (p *parser) subscript() (Segment, error) {
	p.skipSpace()

	var seg Segment
	switch c := p.peek(); {
	case c == '"' || c == '\'':
		key, err := p.str()
		if err != nil {
			return seg, err
		}
		seg = Segment{Field: key}
	case isDigit(c):
		start := p.pos
		for isDigit(p.peek()) {
			p.pos++
		}
		digits := p.src[start:p.pos]
		index, err := strconv.Atoi(digits)
		if err != nil {
			p.pos = start
			return seg, p.errorf("array index %s is too large", digits)
		}
		seg = Segment{Index: index, IsIndex: true}
	case c == '-':
		return seg, p.errorf("array index must not be negative")
	default:
		return seg, p.errorf("unexpected %s, expected an array index or a quoted key", p.describeNext())
	}

	p.skipSpace()
	if !p.consume(']') {
		return seg, p.errorf(`unexpected %s, expected "]"`, p.describeNext())
	}
	return seg, nil
}

// ident parses an identifier; what names the expected thing in the error.
func (p *parser) ident(what string) (string, error) {
	start := p.pos
	if !isIdentStart(p.peek()) {
		return "", p.errorf("unexpected %s, expected %s", p.describeNext(), what)
	}
	for isIdentPart(p.peek()) {
		p.pos++
	}
	return p.src[start:p.pos], nil
}

// number parses a number literal: an optional minus, digits and an optional
// fraction.
func (p *parser) number() (json.Number, error) {
	start := p.pos
	p.consume('-')
	if !isDigit(p.peek()) {
		return "", p.errorf("unexpected %s, expected a digit", p.describeNext())
	}
	for isDigit(p.peek()) {
		p.pos++
	}
	if p.peek() == '.' {
		p.pos++
		if !isDigit(p.peek()) {
			return "", p.errorf("unexpected %s, expected a digit after the decimal point", p.describeNext())
		}
		for isDigit(p.peek()) {
			p.pos++
		}
	}
	return json.Number(p.src[start:p.pos]), nil
}

// str parses a string literal in single or double quotes. A backslash makes
// the next character literal, so quotes and backslashes can be written
// inside the string.
func (p *parser) str() (string, error) {
	quoteChar := p.src[p.pos]
	p.pos++

	var b strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == quoteChar:
			p.pos++
			return b.String(), nil
		case c == '\\' && p.pos+1 < len(p.src):
			b.WriteByte(p.src[p.pos+1])
			p.pos += 2
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
	return "", p.errorf("unclosed string")
}

func (p *parser) peek() byte {
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return 0
}

func (p *parser) consume(c byte) bool {
	if p.pos < len(p.src) && p.src[p.pos] == c {
		p.pos++
		return true
	}
	return false
}

func (p *parser) hasPrefix(prefix string) bool {
	return strings.HasPrefix(p.src[p.pos:], prefix)
}

func (p *parser) skipSpace() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// describeNext describes the next character for an error message.
func (p *parser) describeNext() string {
	if p.pos >= len(p.src) {
		return "end of template"
	}
	// Show the whole character, not the first byte of a multi-byte one.
	for _, r := range p.src[p.pos:] {
		return strconv.QuoteRune(r)
	}
	return "end of template"
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool { return isIdentStart(c) || isDigit(c) }

// isIdent reports whether s is an identifier and so can be written in a path
// as ".s" rather than as a quoted key.
func isIdent(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentPart(s[i]) {
			return false
		}
	}
	return true
}
