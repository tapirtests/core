package extract_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tapirtests/core/extract"
)

func TestParse(t *testing.T) {
	valid := []string{
		"$",
		"$.status",
		"$.body.id",
		"$.body.items[0].name",
		"$.body.items[-1]", // the last element is still one value
		"$.headers['X-Request-Id']",
		`$.headers["Content-Type"]`,
		"$['first-name']",
		`$['it\'s']`,
		"$.body['имя']",
		"$.matrix[1][2]",
	}
	for _, path := range valid {
		t.Run("valid "+path, func(t *testing.T) {
			if _, err := extract.Parse(path); err != nil {
				t.Errorf("Parse(%q): %v", path, err)
			}
		})
	}

	// Valid JSONPath, but it may select many values.
	notExact := []string{
		"$.body.items[*]",
		"$.body.items[*].id",
		"$.body.*",
		"$..id",
		"$.body.items[0:2]",
		"$.body.items[0,1]",
		"$.body['a','b']",
		"$.body.items[?(@.price > 10)]",
		"$.body.items[?@.active].id",
	}
	for _, path := range notExact {
		t.Run("not exact "+path, func(t *testing.T) {
			_, err := extract.Parse(path)
			if !errors.Is(err, extract.ErrNotExact) {
				t.Errorf("Parse(%q): error = %v, want ErrNotExact", path, err)
			}
		})
	}

	malformed := []string{
		"",
		"body.id",    // no root
		"$.",         // nothing after the dot
		"$.body[",    // unclosed bracket
		"$.body.id ", // trailing space
		"$.body..",
		"$['unclosed]",
		"{{productId}}", // a template is not a path
	}
	for _, path := range malformed {
		t.Run("malformed "+path, func(t *testing.T) {
			_, err := extract.Parse(path)
			if err == nil {
				t.Fatalf("Parse(%q): expected an error", path)
			}
			if errors.Is(err, extract.ErrNotExact) {
				t.Errorf("Parse(%q): a syntax error is reported as ErrNotExact", path)
			}
			if strings.HasPrefix(err.Error(), "jsonpath:") {
				t.Errorf("Parse(%q): error %q keeps the library prefix", path, err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	doc := map[string]any{
		"status":  json.Number("201"),
		"headers": map[string]any{"Content-Type": "application/json", "X-Request-Id": "abc"},
		"body": map[string]any{
			"id":        json.Number("57"),
			"name":      "Phone",
			"deletedAt": nil,
			"active":    false,
			"empty":     "",
			"tags":      []any{"new", "sale"},
			"owner":     map[string]any{"first-name": "Ann"},
			"items":     []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}},
		},
	}

	tests := []struct {
		path  string
		want  any
		found bool
	}{
		{"$.status", json.Number("201"), true},
		{"$.headers['X-Request-Id']", "abc", true},
		{"$.body.id", json.Number("57"), true},
		{"$.body.name", "Phone", true},
		{"$.body.tags[1]", "sale", true},
		{"$.body.tags[-1]", "sale", true},
		{"$.body.items[0].name", "a", true},
		{"$.body.owner['first-name']", "Ann", true},

		// Whole objects and arrays can be extracted.
		{"$.body.tags", []any{"new", "sale"}, true},
		{"$.body.owner", map[string]any{"first-name": "Ann"}, true},
		{"$", doc, true},

		// Null, false and empty values exist.
		{"$.body.deletedAt", nil, true},
		{"$.body.active", false, true},
		{"$.body.empty", "", true},

		// Nothing at the path.
		{"$.body.missing", nil, false},
		{"$.body.owner.email", nil, false},
		{"$.body.tags[5]", nil, false},
		{"$.body.name.first", nil, false},     // field of a string
		{"$.body.id[0]", nil, false},          // index of a number
		{"$.body.deletedAt.year", nil, false}, // field of null
		{"$.headers['content-type']", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p, err := extract.Parse(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			got, found := p.Get(doc)
			if found != tt.found {
				t.Fatalf("found = %v, want %v", found, tt.found)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("value = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPathString(t *testing.T) {
	p, err := extract.Parse("$.body.items[0]['first-name']")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := p.String(), `$["body"]["items"][0]["first-name"]`; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
