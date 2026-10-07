package extract_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/tapirtests/core/extract"
	"github.com/tapirtests/core/protocol"
)

func TestDocument(t *testing.T) {
	res := &protocol.Result{
		Status:   201,
		Headers:  map[string]string{"content-type": "application/json", "x-request-id": "abc"},
		Body:     map[string]any{"id": json.Number("57"), "tags": []any{"new"}},
		Duration: 84*time.Millisecond + 600*time.Microsecond,
	}
	doc := extract.Document(res)

	want := map[string]any{
		"status":   json.Number("201"),
		"headers":  map[string]any{"content-type": "application/json", "x-request-id": "abc"},
		"body":     map[string]any{"id": json.Number("57"), "tags": []any{"new"}},
		"duration": json.Number("84"),
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("Document\n got  %#v\n want %#v", doc, want)
	}

	// Every part of the document is reachable by a path.
	paths := map[string]any{
		"$.status":                  json.Number("201"),
		"$.headers['content-type']": "application/json",
		"$.headers['x-request-id']": "abc",
		"$.body.id":                 json.Number("57"),
		"$.body.tags[0]":            "new",
		"$.duration":                json.Number("84"),
	}
	for path, want := range paths {
		p, err := extract.Parse(path)
		if err != nil {
			t.Fatalf("Parse(%q): %v", path, err)
		}
		got, ok := p.Get(doc)
		if !ok || got != want {
			t.Errorf("%s = %#v (found %v), want %#v", path, got, ok, want)
		}
	}
}

func TestDocumentOfEmptyResult(t *testing.T) {
	doc := extract.Document(&protocol.Result{Status: 204})

	// An empty body is null, not a missing field: "$.body exists" is true,
	// while a field inside it is not found.
	body, _ := extract.Parse("$.body")
	if got, ok := body.Get(doc); !ok || got != nil {
		t.Errorf("$.body = %#v (found %v), want null", got, ok)
	}
	field, _ := extract.Parse("$.body.id")
	if _, ok := field.Get(doc); ok {
		t.Error("$.body.id is found in an empty body")
	}
	header, _ := extract.Parse("$.headers['content-type']")
	if _, ok := header.Get(doc); ok {
		t.Error("a header is found in a response without headers")
	}
	if doc["duration"] != json.Number("0") {
		t.Errorf("duration = %#v, want 0", doc["duration"])
	}
}

func TestDocumentPartsMatchConstants(t *testing.T) {
	doc := extract.Document(&protocol.Result{})
	if len(doc) != len(extract.Parts) {
		t.Fatalf("document has %d parts, Parts lists %d", len(doc), len(extract.Parts))
	}
	for _, part := range extract.Parts {
		if _, ok := doc[part]; !ok {
			t.Errorf("document has no part %q", part)
		}
	}
}

func TestStepsAndPart(t *testing.T) {
	tests := []struct {
		path  string
		steps []extract.Step
		part  string
	}{
		{"$", []extract.Step{}, ""},
		{"$.status", []extract.Step{{Field: "status"}}, "status"},
		{"$.body.items[0].name", []extract.Step{{Field: "body"}, {Field: "items"}, {Index: 0, IsIndex: true}, {Field: "name"}}, "body"},
		{"$.headers['content-type']", []extract.Step{{Field: "headers"}, {Field: "content-type"}}, "headers"},
		{"$.body.tags[-1]", []extract.Step{{Field: "body"}, {Field: "tags"}, {Index: -1, IsIndex: true}}, "body"},
		{"$[0].id", []extract.Step{{Index: 0, IsIndex: true}, {Field: "id"}}, ""},
		{"$['body']", []extract.Step{{Field: "body"}}, "body"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p, err := extract.Parse(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got := p.Steps(); !reflect.DeepEqual(got, tt.steps) {
				t.Errorf("Steps() = %+v, want %+v", got, tt.steps)
			}
			if got := p.Part(); got != tt.part {
				t.Errorf("Part() = %q, want %q", got, tt.part)
			}
		})
	}
}
