package testutil

import (
	"testing"

	"github.com/tapirtests/core/model"
)

// Tests rely on ValidProject returning independent copies: breaking one
// project must never leak into another.
func TestValidProjectReturnsIndependentCopies(t *testing.T) {
	a, b := ValidProject(), ValidProject()

	a.Requests["login"].Inputs[0].Name = "changed"
	a.Requests["login"].Responses["200"].Schema[0] = 'X'
	a.Scenarios["login"].Steps[0].(*model.RequestCall).Inputs["body.username"] = "changed"
	a.Root.Main.Groups[0].Name = "changed"
	delete(a.SecuritySchemes, "bearer")

	if b.Requests["login"].Inputs[0].Name != "body.username" {
		t.Error("request inputs are shared between copies")
	}
	if b.Requests["login"].Responses["200"].Schema[0] != '{' {
		t.Error("schemas are shared between copies")
	}
	if b.Scenarios["login"].Steps[0].(*model.RequestCall).Inputs["body.username"] != "{{username}}" {
		t.Error("step inputs are shared between copies")
	}
	if b.Root.Main.Groups[0].Name != "products" {
		t.Error("groups are shared between copies")
	}
	if _, ok := b.SecuritySchemes["bearer"]; !ok {
		t.Error("security schemes are shared between copies")
	}
}
