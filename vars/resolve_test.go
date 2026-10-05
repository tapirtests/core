package vars_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tapirtests/core/random"
	"github.com/tapirtests/core/vars"
)

func num(s string) json.Number { return json.Number(s) }

// fakeRandom is a Caller that returns a fixed value per function and records
// the calls.
type fakeRandom struct {
	calls []string
}

func (f *fakeRandom) Call(name string, args []any) (any, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s%v", name, args))
	switch name {
	case "email":
		return "test-abc@example.com", nil
	case "int":
		return num("42"), nil
	case "bool":
		return true, nil
	default:
		return nil, fmt.Errorf("unknown function random.%s", name)
	}
}

// testResolver returns a resolver over a scope with values of every kind.
//
//	token     "abc"
//	productId 57
//	price     19.99
//	active    true
//	deletedAt null
//	empty     ""
//	world     [{title: "Earth", moons: 1}, {title: "Mars", moons: 2}]
//	product   {id: 57, name: "Phone", tags: ["new", "sale"], owner: {name: "Ann"},
//	           "first-name": "Ann", "2fa": true}
//	matrix    [[1, 2], [3, 4]]
//	login     {token: "seller-token"}      an alias holding scenario outputs
func testResolver() (vars.Resolver, *fakeRandom) {
	s := vars.NewScope()
	s.Set("token", "abc")
	s.Set("productId", num("57"))
	s.Set("price", num("19.99"))
	s.Set("active", true)
	s.Set("deletedAt", nil)
	s.Set("empty", "")
	s.Set("world", []any{
		map[string]any{"title": "Earth", "moons": num("1")},
		map[string]any{"title": "Mars", "moons": num("2")},
	})
	s.Set("product", map[string]any{
		"id":         num("57"),
		"name":       "Phone",
		"tags":       []any{"new", "sale"},
		"owner":      map[string]any{"name": "Ann"},
		"first-name": "Ann",
		"2fa":        true,
	})
	s.Set("matrix", []any{[]any{num("1"), num("2")}, []any{num("3"), num("4")}})
	s.Set("login", map[string]any{"token": "seller-token"})

	env := map[string]string{"BASE_URL": "http://api:8080", "EMPTY": ""}
	rnd := &fakeRandom{}
	return vars.Resolver{
		Scope:  s,
		Env:    func(name string) (string, bool) { v, ok := env[name]; return v, ok },
		Random: rnd,
	}, rnd
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  any
	}{
		// Values without templates are returned as they are.
		{"plain string", "hello", "hello"},
		{"empty string", "", ""},
		{"number", num("5"), num("5")},
		{"bool", true, true},
		{"null", nil, nil},
		{"escaped braces", `\{{token}}`, "{{token}}"},
		{"escaped braces in text", `use \{{name}} here`, "use {{name}} here"},

		// A single expression keeps the type of the value.
		{"string variable", "{{token}}", "abc"},
		{"number variable", "{{productId}}", num("57")},
		{"float variable", "{{price}}", num("19.99")},
		{"bool variable", "{{active}}", true},
		{"null variable", "{{deletedAt}}", nil},
		{"empty string variable", "{{empty}}", ""},
		{"array variable", "{{matrix}}", []any{[]any{num("1"), num("2")}, []any{num("3"), num("4")}}},
		{"object variable", "{{login}}", map[string]any{"token": "seller-token"}},
		{"spaces inside braces", "{{  productId  }}", num("57")},

		// Paths into objects and arrays.
		{"field", "{{product.name}}", "Phone"},
		{"number field", "{{product.id}}", num("57")},
		{"nested field", "{{product.owner.name}}", "Ann"},
		{"index", "{{world[1]}}", map[string]any{"title": "Mars", "moons": num("2")}},
		{"index then field", "{{world[0].title}}", "Earth"},
		{"field then index", "{{product.tags[1]}}", "sale"},
		{"nested indexes", "{{matrix[1][0]}}", num("3")},
		{"quoted key", `{{product["first-name"]}}`, "Ann"},
		{"quoted key starting with a digit", `{{product['2fa']}}`, true},
		{"quoted identifier key", `{{product["name"]}}`, "Phone"},
		{"alias output", "{{login.token}}", "seller-token"},

		// An expression among text gives a string.
		{"text and string", "Bearer {{token}}", "Bearer abc"},
		{"text and number", "id-{{productId}}", "id-57"},
		{"text and float", "{{price}} EUR", "19.99 EUR"},
		{"text and bool", "active={{active}}", "active=true"},
		{"text and null", "/users/{{deletedAt}}", "/users/null"},
		{"text and empty string", "[{{empty}}]", "[]"},
		{"text and array", "tags: {{product.tags}}", `tags: ["new","sale"]`},
		{"text and object", "owner: {{product.owner}}", `owner: {"name":"Ann"}`},
		{"two expressions", "{{productId}}{{productId}}", "5757"},
		{"expression with trailing space", "{{productId}} ", "57 "},
		{"several expressions", "{{product.name}} #{{product.id}} by {{product.owner.name}}", "Phone #57 by Ann"},
		{"escaped and real", `\{{x}} = {{productId}}`, "{{x}} = 57"},

		// Env.
		{"env", "{{env.BASE_URL}}", "http://api:8080"},
		{"env in text", "{{env.BASE_URL}}/products/{{productId}}", "http://api:8080/products/57"},
		{"empty env value", "{{env.EMPTY}}", ""},

		// Random.
		{"random string", "{{random.email}}", "test-abc@example.com"},
		{"random number keeps its type", "{{random.int(1, 999)}}", num("42")},
		{"random bool keeps its type", "{{random.bool}}", true},
		{"random in text", "user-{{random.int(1, 9)}}", "user-42"},

		// Objects and arrays are resolved deeply; keys are not templates.
		{
			"object",
			map[string]any{"name": "{{product.name}}", "id": "{{productId}}", "static": num("1"), "{{token}}": "key"},
			map[string]any{"name": "Phone", "id": num("57"), "static": num("1"), "{{token}}": "key"},
		},
		{
			"array",
			[]any{"{{token}}", num("2"), "x-{{productId}}", nil},
			[]any{"abc", num("2"), "x-57", nil},
		},
		{
			"nested",
			map[string]any{"items": []any{map[string]any{"productId": "{{productId}}", "tags": "{{product.tags}}"}}},
			map[string]any{"items": []any{map[string]any{"productId": num("57"), "tags": []any{"new", "sale"}}}},
		},
		{"empty object", map[string]any{}, map[string]any{}},
		{"empty array", []any{}, []any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := testResolver()
			got, err := r.Resolve(tt.value)
			if err != nil {
				t.Fatalf("Resolve(%#v): %v", tt.value, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve(%#v)\n got  %#v\n want %#v", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolveErrors(t *testing.T) {
	tests := []struct {
		name  string
		value any
		expr  string // expected canonical expression
		at    string // expected place inside the value
		msg   string // expected substring of the message
	}{
		// Variables.
		{"unknown variable", "{{nope}}", "nope", "", `unknown variable "nope"`},
		{"unknown variable in text", "id-{{nope}}", "nope", "", `unknown variable "nope"`},
		{"unknown root of a path", "{{nope.field[0]}}", "nope.field[0]", "", `unknown variable "nope"`},
		{"names are case-sensitive", "{{Token}}", "Token", "", `unknown variable "Token"`},

		// Paths.
		{"missing field", "{{product.price}}", "product.price", "", `product has no field "price"`},
		{"missing nested field", "{{product.owner.email}}", "product.owner.email", "", `product.owner has no field "email"`},
		{"field of a string", "{{token.length}}", "token.length", "", `token is a string, not an object: cannot take field "length"`},
		{"field of a number", "{{productId.value}}", "productId.value", "", "productId is a number, not an object"},
		{"field of null", "{{deletedAt.year}}", "deletedAt.year", "", "deletedAt is null, not an object"},
		{"field of an array", "{{world.title}}", "world.title", "", "world is an array, not an object"},
		{"index of an object", "{{product[0]}}", "product[0]", "", "product is an object, not an array: cannot take element 0"},
		{"index of a string", "{{token[0]}}", "token[0]", "", "token is a string, not an array"},
		{"index out of range", "{{world[2]}}", "world[2]", "", "world has 2 element(s): index 2 is out of range"},
		{"index out of range deep", "{{product.tags[5]}}", "product.tags[5]", "", "product.tags has 2 element(s): index 5 is out of range"},
		{"error in the middle of a path", "{{world[0].name.first}}", "world[0].name.first", "", `world[0] has no field "name"`},
		{"numeric key is not an index", `{{world["0"]}}`, `world["0"]`, "", "world is an array, not an object"},

		// Env and random.
		{"unknown env variable", "{{env.NOPE}}", "env.NOPE", "", `unknown env variable "NOPE"`},
		{"failed call", "{{random.address}}", "random.address", "", "unknown function random.address"},

		// The place of the failing template inside the value.
		{"in an object", map[string]any{"name": "{{nope}}"}, "nope", ".name", `unknown variable "nope"`},
		{"in an array", []any{"ok", "{{nope}}"}, "nope", "[1]", `unknown variable "nope"`},
		{
			"deep in the value",
			map[string]any{"items": []any{map[string]any{"owner-id": "{{nope}}"}}},
			"nope", `.items[0]["owner-id"]`, `unknown variable "nope"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := testResolver()
			_, err := r.Resolve(tt.value)
			if err == nil {
				t.Fatalf("Resolve(%#v): expected an error", tt.value)
			}
			var rerr *vars.ResolveError
			if !errors.As(err, &rerr) {
				t.Fatalf("error is %T (%v), want *vars.ResolveError", err, err)
			}
			if rerr.Expr != tt.expr {
				t.Errorf("Expr = %q, want %q", rerr.Expr, tt.expr)
			}
			if rerr.At != tt.at {
				t.Errorf("At = %q, want %q", rerr.At, tt.at)
			}
			if !strings.Contains(rerr.Msg, tt.msg) {
				t.Errorf("message %q does not contain %q", rerr.Msg, tt.msg)
			}
		})
	}
}

func TestResolveErrorText(t *testing.T) {
	r, _ := testResolver()

	_, err := r.Resolve("id-{{world[5].title}}")
	want := `template "id-{{world[5].title}}": world has 2 element(s): index 5 is out of range`
	if err == nil || err.Error() != want {
		t.Errorf("error = %v\n want  %s", err, want)
	}
	var rerr *vars.ResolveError
	if errors.As(err, &rerr) && rerr.Pos != 3 {
		t.Errorf("Pos = %d, want 3", rerr.Pos)
	}

	_, err = r.Resolve(map[string]any{"owner": map[string]any{"id": "{{nope}}"}})
	want = `template "{{nope}}" (at .owner.id): unknown variable "nope"`
	if err == nil || err.Error() != want {
		t.Errorf("error = %v\n want  %s", err, want)
	}
}

func TestResolveSyntaxError(t *testing.T) {
	r, _ := testResolver()
	for _, value := range []any{"{{token", map[string]any{"a": []any{"{{}}"}}} {
		_, err := r.Resolve(value)
		var perr *vars.ParseError
		if !errors.As(err, &perr) {
			t.Errorf("Resolve(%#v): error is %T (%v), want *vars.ParseError", value, err, err)
		}
	}
}

// The model is read-only: resolving a step must not change its inputs.
func TestResolveDoesNotModifyInput(t *testing.T) {
	r, _ := testResolver()
	input := map[string]any{
		"name": "{{product.name}}",
		"tags": []any{"{{token}}", map[string]any{"id": "{{productId}}"}},
	}
	before := fmt.Sprintf("%#v", input)

	if _, err := r.Resolve(input); err != nil {
		t.Fatal(err)
	}
	if after := fmt.Sprintf("%#v", input); after != before {
		t.Errorf("input changed:\n before %s\n after  %s", before, after)
	}
}

func TestResolveMissingContext(t *testing.T) {
	tests := []struct {
		name  string
		value string
		msg   string
	}{
		{"no scope", "{{token}}", "variables are not available here"},
		{"no env", "{{env.BASE_URL}}", "env variables are not available here"},
		{"no random", "{{random.email}}", "random values are not available here"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vars.Resolver{}.Resolve(tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.msg) {
				t.Errorf("error = %v, want one containing %q", err, tt.msg)
			}
		})
	}

	// Plain text needs no context at all.
	got, err := vars.Resolver{}.Resolve("plain")
	if err != nil || got != "plain" {
		t.Errorf("Resolve(plain) = %v, %v", got, err)
	}
}

// Every evaluation of a random expression is a new call.
func TestResolveCallsRandomPerExpression(t *testing.T) {
	r, rnd := testResolver()
	if _, err := r.Resolve([]any{"{{random.email}}", "{{random.int(1, 9)}}-{{random.int(1, 9)}}"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"email[]", "int[1 9]", "int[1 9]"}
	if !reflect.DeepEqual(rnd.calls, want) {
		t.Errorf("calls = %v, want %v", rnd.calls, want)
	}
}

// Variables are looked up through the scope chain, nearest first.
func TestResolveUsesScopeChain(t *testing.T) {
	root := vars.NewScope()
	root.Set("sellerToken", "root-token")
	root.Set("name", "from-root")
	scenario := root.Child().Child()
	scenario.Set("name", "from-scenario")

	r := vars.Resolver{Scope: scenario}
	got, err := r.Resolve("{{sellerToken}}/{{name}}")
	if err != nil || got != "root-token/from-scenario" {
		t.Errorf("Resolve = %v, %v", got, err)
	}
}

// The real generator plugs in as the Caller, with its own argument checks.
func TestResolveWithRealRandom(t *testing.T) {
	r := vars.Resolver{Scope: vars.NewScope(), Random: random.New(81723)}

	got, err := r.Resolve(map[string]any{"email": "{{random.email}}", "price": "{{random.int(1, 999)}}"})
	if err != nil {
		t.Fatal(err)
	}
	body := got.(map[string]any)
	if email, ok := body["email"].(string); !ok || !strings.HasSuffix(email, "@example.com") {
		t.Errorf("email = %#v", body["email"])
	}
	if _, ok := body["price"].(json.Number); !ok {
		t.Errorf("price = %#v, want a number", body["price"])
	}

	_, err = r.Resolve("{{random.int(10, 1)}}")
	if err == nil || !strings.Contains(err.Error(), "min 10 is greater than max 1") {
		t.Errorf("error = %v", err)
	}
}
