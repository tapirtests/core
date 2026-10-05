package vars_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/tapirtests/core/vars"
)

// describe renders the structure of a template so that tests can compare it
// as one string: text parts as T("...") and expressions as their kind with
// the canonical form, e.g. `T("id-") var(product.tags[0])`.
func describe(t *vars.Template) string {
	var parts []string
	for _, p := range t.Parts() {
		if p.Expr == nil {
			parts = append(parts, fmt.Sprintf("T(%q)", p.Text))
			continue
		}
		kind := map[vars.ExprKind]string{vars.ExprVar: "var", vars.ExprEnv: "env", vars.ExprCall: "call"}[p.Expr.Kind]
		parts = append(parts, fmt.Sprintf("%s(%s)", kind, p.Expr))
	}
	return strings.Join(parts, " ")
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		// Plain text.
		{"empty", "", ""},
		{"text", "hello, world", `T("hello, world")`},
		{"single braces are text", "a {b} c", `T("a {b} c")`},
		{"closing braces alone are text", "a }} b", `T("a }} b")`},
		{"backslash alone is text", `C:\temp\file`, `T("C:\\temp\\file")`},

		// Variables.
		{"variable", "{{productId}}", "var(productId)"},
		{"variable with spaces", "{{  productId  }}", "var(productId)"},
		{"variable with newlines", "{{\n\tproductId\n}}", "var(productId)"},
		{"underscore and digits", "{{_item_2}}", "var(_item_2)"},
		{"text around", "id-{{productId}}-x", `T("id-") var(productId) T("-x")`},
		{"two expressions", "{{a}}{{b}}", "var(a) var(b)"},
		{"expression between text", "{{a}}, {{b}}!", `var(a) T(", ") var(b) T("!")`},

		// Paths.
		{"field", "{{product.name}}", "var(product.name)"},
		{"nested fields", "{{a.b.c}}", "var(a.b.c)"},
		{"index", "{{items[0]}}", "var(items[0])"},
		{"index with spaces", "{{items[ 12 ]}}", "var(items[12])"},
		{"index then field", "{{world[0].title}}", "var(world[0].title)"},
		{"field then index", "{{product.tags[1]}}", "var(product.tags[1])"},
		{"nested indexes", "{{matrix[1][2]}}", "var(matrix[1][2])"},
		{"long path", "{{a.b[2].c.d[0]}}", "var(a.b[2].c.d[0])"},
		{"alias output", "{{productLifecycle.productId}}", "var(productLifecycle.productId)"},
		{"quoted key", `{{headers["Content-Type"]}}`, `var(headers["Content-Type"])`},
		{"single-quoted key", `{{headers['Content-Type']}}`, `var(headers["Content-Type"])`},
		{"quoted identifier key is normalized", `{{product["name"]}}`, "var(product.name)"},
		{"quoted key with escapes", `{{m["a\"b\\c"]}}`, `var(m["a\"b\\c"])`},
		{"quoted key with braces", `{{m["}}"]}}`, `var(m["}}"])`},
		{"quoted key with unicode", `{{m["имя"]}}`, `var(m["имя"])`},
		{"empty quoted key", `{{m[""]}}`, `var(m[""])`},

		// Env.
		{"env", "{{env.BASE_URL}}", "env(env.BASE_URL)"},
		{"env in text", "{{env.BASE_URL}}/products", `env(env.BASE_URL) T("/products")`},
		{"variable named like env prefix", "{{environment}}", "var(environment)"},

		// Random.
		{"call without parentheses", "{{random.email}}", "call(random.email)"},
		{"call with empty parentheses", "{{random.uuid()}}", "call(random.uuid)"},
		{"call with numbers", "{{random.int(1, 999)}}", "call(random.int(1, 999))"},
		{"call with spaces", "{{ random.int( 1 ,999 ) }}", "call(random.int(1, 999))"},
		{"call with negative and float", "{{random.float(-1.5, 2)}}", "call(random.float(-1.5, 2))"},
		{"call with string", `{{random.pick("a", 'b')}}`, `call(random.pick("a", "b"))`},
		{"call with string containing braces", `{{random.pick("}}")}}`, `call(random.pick("}}"))`},

		// Escaping.
		{"escaped braces", `\{{name}}`, `T("{{name}}")`},
		{"escaped braces in text", `a \{{b}} c`, `T("a {{b}} c")`},
		{"escaped then real", `\{{a}} {{b}}`, `T("{{a}} ") var(b)`},
		{"escape applies to opening braces only", `{{a}}\}}`, `var(a) T("\\}}")`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := vars.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.src, err)
			}
			if got := describe(tmpl); got != tt.want {
				t.Errorf("Parse(%q)\n got  %s\n want %s", tt.src, got, tt.want)
			}
			if tmpl.Source() != tt.src {
				t.Errorf("Source() = %q, want %q", tmpl.Source(), tt.src)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		pos  int    // expected offset of the error
		msg  string // expected substring of the message
	}{
		// Braces.
		{"unclosed", "{{name", 0, `unclosed "{{"`},
		{"unclosed after text", "id-{{name", 3, `unclosed "{{"`},
		{"unclosed with one brace", "{{name}", 6, `expected "}}"`},
		{"unclosed in path", "{{a.", 0, `unclosed "{{"`},
		{"unclosed in subscript", "{{a[0", 0, `unclosed "{{"`},
		{"unclosed in string", `{{m["key`, 0, `unclosed "{{"`},
		{"unclosed in call", "{{random.int(1,", 0, `unclosed "{{"`},
		{"second expression unclosed", "{{a}} {{b", 6, `unclosed "{{"`},
		{"empty", "{{}}", 2, "empty expression"},
		{"empty with spaces", "{{   }}", 5, "empty expression"},
		{"nested", "{{a {{b}} }}", 4, `expected "}}"`},
		{"three braces", "{{{a}}", 2, "expected a name"},

		// Names.
		{"starts with digit", "{{1abc}}", 2, "expected a name"},
		{"dash in name", "{{product-id}}", 9, `expected "}}"`},
		{"jsonpath is not a template path", "{{$.body.id}}", 2, "expected a name"},
		{"space inside path", "{{a .b}}", 4, `expected "}}"`},
		{"two names", "{{a b}}", 4, `expected "}}"`},
		{"non-ASCII name", "{{имя}}", 2, "expected a name"},

		// Paths.
		{"trailing dot", "{{a.}}", 4, "expected a field name"},
		{"double dot", "{{a..b}}", 4, "expected a field name"},
		{"empty subscript", "{{a[]}}", 4, "expected an array index or a quoted key"},
		{"negative index", "{{a[-1]}}", 4, "must not be negative"},
		{"float index", "{{a[1.5]}}", 5, `expected "]"`},
		{"unquoted key", "{{a[name]}}", 4, "expected an array index or a quoted key"},
		{"unclosed subscript", "{{a[0}}", 5, `expected "]"`},
		{"huge index", "{{a[99999999999999999999]}}", 4, "too large"},
		{"unclosed key string", `{{a["key]}}`, 0, `unclosed "{{"`},

		// Env.
		{"env alone", "{{env}}", 5, `expected ".NAME"`},
		{"env without name", "{{env.}}", 6, "expected an env variable name"},
		{"env with field", "{{env.URL.host}}", 9, "has no fields or elements"},
		{"env with index", "{{env.URL[0]}}", 9, "has no fields or elements"},

		// Calls.
		{"random alone", "{{random}}", 8, `expected ".function"`},
		{"random without function", "{{random.}}", 9, "expected a function name"},
		{"call of a variable", "{{getId()}}", 7, "only random.* functions can be called"},
		{"call of a variable path", "{{a.b(1)}}", 5, "only random.* functions can be called"},
		{"path after call", "{{random.email.domain}}", 14, "cannot be followed by a path"},
		{"unclosed arguments", "{{random.int(1, 2}}", 17, `expected "," or ")"`},
		{"missing comma", "{{random.int(1 2)}}", 15, `expected "," or ")"`},
		{"trailing comma", "{{random.int(1,)}}", 15, "expected a number or a string"},
		{"variable as argument", "{{random.int(min, 9)}}", 13, "expected a number or a string"},
		{"minus without digits", "{{random.int(-)}}", 14, "expected a digit"},
		{"dot without fraction", "{{random.int(1.)}}", 15, "expected a digit after the decimal point"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vars.Parse(tt.src)
			if err == nil {
				t.Fatalf("Parse(%q): expected an error", tt.src)
			}
			var perr *vars.ParseError
			if !errors.As(err, &perr) {
				t.Fatalf("Parse(%q): error is %T, want *vars.ParseError", tt.src, err)
			}
			if perr.Pos != tt.pos {
				t.Errorf("Parse(%q): error at offset %d, want %d (%v)", tt.src, perr.Pos, tt.pos, err)
			}
			if !strings.Contains(perr.Msg, tt.msg) {
				t.Errorf("Parse(%q): message %q does not contain %q", tt.src, perr.Msg, tt.msg)
			}
			if perr.Src != tt.src {
				t.Errorf("Parse(%q): error source is %q", tt.src, perr.Src)
			}
		})
	}
}

func TestExprFields(t *testing.T) {
	tmpl, err := vars.Parse(`x {{world[0]["the title"].a}} {{env.HOST}} {{random.int(1, "a")}}`)
	if err != nil {
		t.Fatal(err)
	}
	exprs := tmpl.Exprs()
	if len(exprs) != 3 {
		t.Fatalf("got %d expressions, want 3", len(exprs))
	}

	v := exprs[0]
	if v.Kind != vars.ExprVar || v.Name != "world" || v.Pos != 2 {
		t.Errorf("variable: kind %v, name %q, pos %d", v.Kind, v.Name, v.Pos)
	}
	wantPath := []vars.Segment{{Index: 0, IsIndex: true}, {Field: "the title"}, {Field: "a"}}
	if len(v.Path) != len(wantPath) {
		t.Fatalf("path = %v, want %v", v.Path, wantPath)
	}
	for i, seg := range v.Path {
		if seg != wantPath[i] {
			t.Errorf("path[%d] = %+v, want %+v", i, seg, wantPath[i])
		}
	}

	if e := exprs[1]; e.Kind != vars.ExprEnv || e.Name != "HOST" || len(e.Path) != 0 {
		t.Errorf("env: kind %v, name %q, path %v", e.Kind, e.Name, e.Path)
	}

	c := exprs[2]
	if c.Kind != vars.ExprCall || c.Name != "int" {
		t.Errorf("call: kind %v, name %q", c.Kind, c.Name)
	}
	if len(c.Args) != 2 || c.Args[0] != json.Number("1") || c.Args[1] != "a" {
		t.Errorf("call args = %#v", c.Args)
	}
}

func TestTemplateKinds(t *testing.T) {
	tests := []struct {
		src    string
		static bool
		single bool
	}{
		{"", true, false},
		{"plain text", true, false},
		{`\{{escaped}}`, true, false},
		{"{{productId}}", false, true},
		{"{{ productId }}", false, true},
		{"{{a.b[0]}}", false, true},
		{"{{random.email}}", false, true},
		{"id-{{productId}}", false, false},
		{"{{productId}} ", false, false},
		{"{{a}}{{b}}", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			tmpl, err := vars.Parse(tt.src)
			if err != nil {
				t.Fatal(err)
			}
			if got := tmpl.IsStatic(); got != tt.static {
				t.Errorf("IsStatic() = %v, want %v", got, tt.static)
			}
			if _, got := tmpl.Single(); got != tt.single {
				t.Errorf("Single() = %v, want %v", got, tt.single)
			}
		})
	}
}

func TestTemplateString(t *testing.T) {
	tests := []struct{ src, want string }{
		{"hello", "hello"},
		{"{{  a.b[ 0 ]  }}", "{{a.b[0]}}"},
		{`{{m['x y']}}`, `{{m["x y"]}}`},
		{`{{m["name"]}}`, "{{m.name}}"},
		{`a \{{b}} {{ c }}`, `a \{{b}} {{c}}`},
		{"{{random.int( 1,2 )}}", "{{random.int(1, 2)}}"},
		{"{{random.uuid()}}", "{{random.uuid}}"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			tmpl, err := vars.Parse(tt.src)
			if err != nil {
				t.Fatal(err)
			}
			got := tmpl.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
			// The canonical form parses to the same canonical form.
			again, err := vars.Parse(got)
			if err != nil {
				t.Fatalf("canonical form %q does not parse: %v", got, err)
			}
			if again.String() != got {
				t.Errorf("canonical form is not stable: %q -> %q", got, again.String())
			}
		})
	}
}

// FuzzParse checks that the parser never panics on arbitrary input and that
// the canonical form of whatever it accepts is stable: it parses again to
// the same structure.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"", "text", "{{a}}", "{{ a.b[0].c }}", `{{m["k"]}}`, "{{env.X}}", "{{random.int(1, 2)}}",
		`\{{a}}`, "{{", "}}", "{{}}", "{{a", "{{a[", `{{a["`, "{{random.f(", "{{a}}{{b}}", "{{{a}}}",
		`{{m['\'']}}`, "{{a[99999999999999999999]}}", "{{имя}}", "\\", `\{`, `{{a}}\`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		tmpl, err := vars.Parse(src)
		if err != nil {
			var perr *vars.ParseError
			if !errors.As(err, &perr) {
				t.Fatalf("Parse(%q): error is %T, want *vars.ParseError", src, err)
			}
			if perr.Pos < 0 || perr.Pos > len(src) {
				t.Fatalf("Parse(%q): error offset %d is out of range", src, perr.Pos)
			}
			return
		}

		canonical := tmpl.String()
		again, err := vars.Parse(canonical)
		if err != nil {
			t.Fatalf("Parse(%q) ok, but its canonical form %q does not parse: %v", src, canonical, err)
		}
		if describe(again) != describe(tmpl) {
			t.Fatalf("Parse(%q): canonical form %q changes the structure\n before %s\n after  %s",
				src, canonical, describe(tmpl), describe(again))
		}
	})
}
