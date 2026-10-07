package assert_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tapirtests/core/assert"
	"github.com/tapirtests/core/extract"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/protocol"
)

func num(s string) json.Number { return json.Number(s) }

// response is the document every case is checked against.
//
//	status   201
//	duration 84 (ms)
//	headers  content-type: application/json; charset=utf-8, x-count: 3
//	body     see below
func response() map[string]any {
	return extract.Document(&protocol.Result{
		Status:   201,
		Duration: 84 * time.Millisecond,
		Headers: map[string]string{
			"content-type": "application/json; charset=utf-8",
			"x-count":      "3",
		},
		Body: map[string]any{
			"id":        num("57"),
			"name":      "Phone",
			"price":     num("19.99"),
			"big":       num("9007199254740993"), // not representable as float64
			"active":    true,
			"deletedAt": nil,
			"empty":     "",
			"emoji":     "héllo 👋",
			"tags":      []any{"new", "sale"},
			"sizes":     []any{num("1"), num("2"), num("3")},
			"nothing":   []any{},
			"owner":     map[string]any{"id": num("7"), "name": "Ann"},
			"items":     []any{map[string]any{"id": num("1")}, map[string]any{"id": num("2")}},
		},
	})
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		op     model.AssertOp
		value  any
		passed bool
	}{
		// equals.
		{"status equals", "$.status", model.OpEquals, num("201"), true},
		{"status differs", "$.status", model.OpEquals, num("200"), false},
		{"number equals with another notation", "$.status", model.OpEquals, num("201.0"), true},
		{"number equals a Go int", "$.status", model.OpEquals, 201, true},
		{"number equals with exponent", "$.body.id", model.OpEquals, num("5.7e1"), true},
		{"float equals", "$.body.price", model.OpEquals, num("19.99"), true},
		{"float differs in the last digit", "$.body.price", model.OpEquals, num("19.990001"), false},
		{"large integers are compared exactly", "$.body.big", model.OpEquals, num("9007199254740992"), false},
		{"string equals", "$.body.name", model.OpEquals, "Phone", true},
		{"string is case-sensitive", "$.body.name", model.OpEquals, "phone", false},
		{"bool equals", "$.body.active", model.OpEquals, true, true},
		{"null equals null", "$.body.deletedAt", model.OpEquals, nil, true},
		{"null is not an empty string", "$.body.deletedAt", model.OpEquals, "", false},
		{"empty string equals empty string", "$.body.empty", model.OpEquals, "", true},
		{"number is not its string", "$.body.id", model.OpEquals, "57", false},
		{"string is not a number", "$.headers['x-count']", model.OpEquals, num("3"), false},
		{"bool is not a string", "$.body.active", model.OpEquals, "true", false},
		{"array equals", "$.body.tags", model.OpEquals, []any{"new", "sale"}, true},
		{"array order matters", "$.body.tags", model.OpEquals, []any{"sale", "new"}, false},
		{"array length matters", "$.body.tags", model.OpEquals, []any{"new"}, false},
		{"array of numbers in another notation", "$.body.sizes", model.OpEquals, []any{1, num("2.0"), num("3")}, true},
		{"object equals", "$.body.owner", model.OpEquals, map[string]any{"name": "Ann", "id": num("7")}, true},
		{"object with an extra field", "$.body.owner", model.OpEquals, map[string]any{"id": num("7"), "name": "Ann", "x": nil}, false},
		{"object with a missing field", "$.body.owner", model.OpEquals, map[string]any{"id": num("7")}, false},
		{"object with a different value", "$.body.owner", model.OpEquals, map[string]any{"id": num("8"), "name": "Ann"}, false},
		{"nested value", "$.body.items[1].id", model.OpEquals, num("2"), true},
		{"header equals", "$.headers['x-count']", model.OpEquals, "3", true},
		{"missing value does not equal", "$.body.missing", model.OpEquals, nil, false},

		// notEquals.
		{"notEquals holds", "$.status", model.OpNotEquals, num("500"), true},
		{"notEquals fails", "$.status", model.OpNotEquals, num("201"), false},
		{"notEquals across kinds", "$.body.id", model.OpNotEquals, "57", true},
		{"notEquals null on a value", "$.body.name", model.OpNotEquals, nil, true},
		{"notEquals null on null", "$.body.deletedAt", model.OpNotEquals, nil, false},
		{"notEquals on a missing value fails", "$.body.missing", model.OpNotEquals, "x", false},

		// in.
		{"in holds", "$.status", model.OpIn, []any{num("200"), num("201")}, true},
		{"in fails", "$.status", model.OpIn, []any{num("200"), num("204")}, false},
		{"in with an empty list", "$.status", model.OpIn, []any{}, false},
		{"in with mixed kinds", "$.body.name", model.OpIn, []any{num("1"), "Phone", nil}, true},
		{"in with null", "$.body.deletedAt", model.OpIn, []any{nil, "never"}, true},
		{"in on a missing value", "$.body.missing", model.OpIn, []any{nil}, false},

		// exists and notExists.
		{"exists", "$.body.id", model.OpExists, nil, true},
		{"null exists", "$.body.deletedAt", model.OpExists, nil, true},
		{"empty string exists", "$.body.empty", model.OpExists, nil, true},
		{"empty array exists", "$.body.nothing", model.OpExists, nil, true},
		{"missing field does not exist", "$.body.missing", model.OpExists, nil, false},
		{"index out of range does not exist", "$.body.tags[5]", model.OpExists, nil, false},
		{"header exists", "$.headers['content-type']", model.OpExists, nil, true},
		{"missing header does not exist", "$.headers['x-debug']", model.OpExists, nil, false},
		{"notExists on a missing field", "$.body.missing", model.OpNotExists, nil, true},
		{"notExists on a present field", "$.body.id", model.OpNotExists, nil, false},
		{"notExists on null", "$.body.deletedAt", model.OpNotExists, nil, false},
		{"exists ignores the value", "$.body.id", model.OpExists, "ignored", true},

		// contains.
		{"string contains", "$.body.name", model.OpContains, "hon", true},
		{"string does not contain", "$.body.name", model.OpContains, "Tablet", false},
		{"string contains is case-sensitive", "$.body.name", model.OpContains, "phone", false},
		{"string contains the empty string", "$.body.name", model.OpContains, "", true},
		{"header contains", "$.headers['content-type']", model.OpContains, "json", true},
		{"string cannot contain a number", "$.headers['x-count']", model.OpContains, num("3"), false},
		{"array contains an element", "$.body.tags", model.OpContains, "sale", true},
		{"array does not contain", "$.body.tags", model.OpContains, "old", false},
		{"array contains a number in another notation", "$.body.sizes", model.OpContains, num("2.0"), true},
		{"array contains an object", "$.body.items", model.OpContains, map[string]any{"id": num("2")}, true},
		{"array contains matches whole elements only", "$.body.tags", model.OpContains, "sal", false},
		{"empty array contains nothing", "$.body.nothing", model.OpContains, "x", false},
		{"number contains nothing", "$.body.id", model.OpContains, "5", false},
		{"object contains nothing", "$.body.owner", model.OpContains, "id", false},
		{"null contains nothing", "$.body.deletedAt", model.OpContains, "x", false},

		// matches.
		{"matches", "$.body.name", model.OpMatches, "^[A-Z][a-z]+$", true},
		{"does not match", "$.body.name", model.OpMatches, "^[0-9]+$", false},
		{"matches a part of the string", "$.headers['content-type']", model.OpMatches, "json", true},
		{"matches unicode", "$.body.emoji", model.OpMatches, `^h.llo`, true},
		{"number does not match", "$.body.id", model.OpMatches, "57", false},
		{"null does not match", "$.body.deletedAt", model.OpMatches, ".*", false},
		{"missing value does not match", "$.body.missing", model.OpMatches, ".*", false},

		// Ordering.
		{"lt holds", "$.duration", model.OpLt, num("500"), true},
		{"lt fails on equal", "$.duration", model.OpLt, num("84"), false},
		{"lt fails on greater", "$.duration", model.OpLt, num("10"), false},
		{"lte holds on equal", "$.duration", model.OpLte, num("84"), true},
		{"lte fails", "$.duration", model.OpLte, num("83"), false},
		{"gt holds", "$.status", model.OpGt, num("200"), true},
		{"gt fails on equal", "$.status", model.OpGt, num("201"), false},
		{"gte holds on equal", "$.status", model.OpGte, num("201"), true},
		{"gte fails", "$.status", model.OpGte, num("202"), false},
		{"float against integer", "$.body.price", model.OpLt, num("20"), true},
		{"float against float", "$.body.price", model.OpGt, num("19.989"), true},
		{"negative bound", "$.body.id", model.OpGt, num("-1"), true},
		{"Go int bound", "$.body.id", model.OpLte, 57, true},
		{"large integers are ordered exactly", "$.body.big", model.OpGt, num("9007199254740992"), true},
		{"string is not ordered", "$.headers['x-count']", model.OpLt, num("10"), false},
		{"null is not ordered", "$.body.deletedAt", model.OpLt, num("10"), false},
		{"missing value is not ordered", "$.body.missing", model.OpLt, num("10"), false},

		// length.
		{"array length", "$.body.tags", model.OpLength, num("2"), true},
		{"array length differs", "$.body.tags", model.OpLength, num("3"), false},
		{"empty array length", "$.body.nothing", model.OpLength, num("0"), true},
		{"string length", "$.body.name", model.OpLength, num("5"), true},
		{"string length counts characters, not bytes", "$.body.emoji", model.OpLength, num("7"), true},
		{"empty string length", "$.body.empty", model.OpLength, num("0"), true},
		{"object length counts fields", "$.body.owner", model.OpLength, num("2"), true},
		{"length written as a float", "$.body.tags", model.OpLength, num("2.0"), true},
		{"number has no length", "$.body.id", model.OpLength, num("2"), false},
		{"null has no length", "$.body.deletedAt", model.OpLength, num("0"), false},
		{"missing value has no length", "$.body.missing", model.OpLength, num("0"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := model.Assertion{Path: tt.path, Op: tt.op, Value: tt.value}
			res, err := assert.Check(a, response())
			if err != nil {
				t.Fatalf("Check: unexpected error: %v", err)
			}
			if res.Passed != tt.passed {
				t.Errorf("Passed = %v, want %v (message: %q)", res.Passed, tt.passed, res.Message)
			}
			if res.Passed != (res.Message == "") {
				t.Errorf("Passed = %v with message %q: a message is set exactly when the check fails", res.Passed, res.Message)
			}
			if res.Path != tt.path || res.Op != tt.op {
				t.Errorf("result repeats %s %s, want %s %s", res.Path, res.Op, tt.path, tt.op)
			}
		})
	}
}

// A malformed assertion is an error, not a failed check: the test is wrong,
// not the API. It is reported even when the response has nothing at the path.
func TestCheckErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		op      model.AssertOp
		value   any
		wantErr string
	}{
		{"malformed path", "$.body.items[", model.OpExists, nil, "assertion path"},
		{"path with a wildcard", "$.body.items[*].id", model.OpExists, nil, "not supported yet"},
		{"empty path", "", model.OpExists, nil, "assertion path"},
		{"unknown operator", "$.status", "between", num("1"), `unknown assertion operator "between"`},
		{"empty operator", "$.status", "", num("1"), "unknown assertion operator"},

		{"in with a string", "$.status", model.OpIn, "200,201", "must be an array, got a string"},
		{"in with null", "$.status", model.OpIn, nil, "must be an array, got null"},
		{"matches with a number", "$.body.name", model.OpMatches, num("5"), "must be a regular expression string, got a number"},
		{"matches with an invalid regexp", "$.body.name", model.OpMatches, "(unclosed", "invalid regular expression"},
		{"lt with a string", "$.duration", model.OpLt, "fast", "must be a number, got a string"},
		{"gte with null", "$.duration", model.OpGte, nil, "must be a number, got null"},
		{"lt with a malformed number", "$.duration", model.OpLt, num("1e"), "must be a number"},
		{"length with a string", "$.body.tags", model.OpLength, "two", "non-negative integer"},
		{"length with a fraction", "$.body.tags", model.OpLength, num("1.5"), "non-negative integer"},
		{"length with a negative number", "$.body.tags", model.OpLength, num("-1"), "non-negative integer"},

		// The expected value is checked even when there is nothing to
		// compare it with.
		{"invalid regexp on a missing value", "$.body.missing", model.OpMatches, "(unclosed", "invalid regular expression"},
		{"in with a string on a missing value", "$.body.missing", model.OpIn, "x", "must be an array"},
		{"lt with a string on a missing value", "$.body.missing", model.OpLt, "x", "must be a number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := assert.Check(model.Assertion{Path: tt.path, Op: tt.op, Value: tt.value}, response())
			if err == nil {
				t.Fatalf("Check: expected an error, got Passed = %v, message %q", res.Passed, res.Message)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
			if res.Passed {
				t.Error("result of a failed evaluation is marked as passed")
			}
		})
	}
}

func TestCheckResultValues(t *testing.T) {
	t.Run("found value", func(t *testing.T) {
		res, _ := assert.Check(model.Assertion{Path: "$.body.id", Op: model.OpEquals, Value: num("58")}, response())
		if !res.Found || res.Actual != num("57") || res.Expected != num("58") {
			t.Errorf("Found = %v, Actual = %#v, Expected = %#v", res.Found, res.Actual, res.Expected)
		}
	})
	t.Run("null value is found", func(t *testing.T) {
		res, _ := assert.Check(model.Assertion{Path: "$.body.deletedAt", Op: model.OpEquals, Value: "x"}, response())
		if !res.Found || res.Actual != nil {
			t.Errorf("Found = %v, Actual = %#v, want a found null", res.Found, res.Actual)
		}
	})
	t.Run("missing value", func(t *testing.T) {
		res, _ := assert.Check(model.Assertion{Path: "$.body.missing", Op: model.OpEquals, Value: "x"}, response())
		if res.Found || res.Actual != nil {
			t.Errorf("Found = %v, Actual = %#v, want nothing found", res.Found, res.Actual)
		}
	})
	t.Run("exists has no expected value", func(t *testing.T) {
		res, _ := assert.Check(model.Assertion{Path: "$.body.id", Op: model.OpExists, Value: "ignored"}, response())
		if res.Expected != nil {
			t.Errorf("Expected = %#v, want nil", res.Expected)
		}
	})
}

func TestCheckMessages(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		op    model.AssertOp
		value any
		want  string
	}{
		{"equals", "$.status", model.OpEquals, num("200"),
			"expected $.status to equal 200, got 201"},
		{"equals across kinds names the kinds", "$.body.id", model.OpEquals, "57",
			`expected $.body.id to equal "57", got 57 (a number, expected a string)`},
		{"equals null", "$.body.name", model.OpEquals, nil,
			`expected $.body.name to equal null, got "Phone" (a string, expected null)`},
		{"notEquals", "$.status", model.OpNotEquals, num("201"),
			"expected $.status not to equal 201, but it is equal"},
		{"in", "$.status", model.OpIn, []any{num("200"), num("204")},
			"expected $.status to be one of [200,204], got 201"},
		{"exists", "$.body.missing", model.OpExists, nil,
			"expected $.body.missing to exist, but there is nothing at this path"},
		{"notExists", "$.body.id", model.OpNotExists, nil,
			"expected $.body.id not to exist, got 57"},
		{"contains", "$.body.tags", model.OpContains, "old",
			`expected $.body.tags to contain "old", got ["new","sale"]`},
		{"contains on a number", "$.body.id", model.OpContains, "5",
			`expected $.body.id to contain "5", but it is a number, not a string or an array: 57`},
		{"matches", "$.body.name", model.OpMatches, "^[0-9]+$",
			`expected $.body.name to match "^[0-9]+$", got "Phone"`},
		{"lt", "$.duration", model.OpLt, num("50"),
			"expected $.duration to be less than 50, got 84"},
		{"gte", "$.status", model.OpGte, num("300"),
			"expected $.status to be at least 300, got 201"},
		{"lt on a string", "$.headers['x-count']", model.OpLt, num("10"),
			`expected $.headers['x-count'] to be less than 10, but it is a string, not a number: "3"`},
		{"length", "$.body.tags", model.OpLength, num("3"),
			"expected $.body.tags to have length 3, got length 2"},
		{"length on a number", "$.body.id", model.OpLength, num("2"),
			"expected $.body.id to have length 2, but it is a number, which has no length: 57"},
		{"missing value", "$.body.missing", model.OpEquals, num("1"),
			"expected $.body.missing to equal 1, but there is nothing at this path"},
		{"missing value with lt", "$.body.missing", model.OpLt, num("1"),
			"expected $.body.missing to be less than 1, but there is nothing at this path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := assert.Check(model.Assertion{Path: tt.path, Op: tt.op, Value: tt.value}, response())
			if err != nil {
				t.Fatal(err)
			}
			if res.Message != tt.want {
				t.Errorf("message\n got  %s\n want %s", res.Message, tt.want)
			}
		})
	}
}

// A message must stay readable when the actual value is a whole response
// body; the full value is in the result, and so in the report.
func TestCheckMessageIsTruncated(t *testing.T) {
	doc := map[string]any{"body": strings.Repeat("я", 5000)}
	res, err := assert.Check(model.Assertion{Path: "$.body", Op: model.OpEquals, Value: "x"}, doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Message) > 400 {
		t.Errorf("message is %d bytes long", len(res.Message))
	}
	if !strings.Contains(res.Message, "…") {
		t.Errorf("message does not show that the value is cut: %q", res.Message)
	}
	if !strings.HasPrefix(res.Message, `expected $.body to equal "x", got "яяя`) {
		t.Errorf("message = %q", res.Message)
	}
	if got, ok := res.Actual.(string); !ok || len([]rune(got)) != 5000 {
		t.Error("the actual value in the result is not the full one")
	}
}
