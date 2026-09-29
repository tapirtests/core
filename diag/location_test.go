package diag_test

import (
	"testing"

	"github.com/tapirtests/core/diag"
)

func TestPointer(t *testing.T) {
	tests := []struct {
		name string
		got  diag.Pointer
		want diag.Pointer
	}{
		{"root", diag.Root, ""},
		{"key", diag.Root.Key("scenarios"), "/scenarios"},
		{"nested", diag.Root.Key("scenarios").Key("login").Key("steps").Index(2), "/scenarios/login/steps/2"},
		{"escape tilde", diag.Root.Key("a~b"), "/a~0b"},
		{"escape slash", diag.Root.Key("GET /pets/{id}"), "/GET ~1pets~1{id}"},
		{"escape order", diag.Root.Key("~1"), "/~01"},
		{"empty key", diag.Root.Key(""), "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestPointerIsImmutable(t *testing.T) {
	base := diag.Root.Key("scenarios")
	_ = base.Key("a")
	_ = base.Key("b")
	if base != "/scenarios" {
		t.Errorf("base pointer changed: %q", base)
	}
}

func TestLocationString(t *testing.T) {
	tests := []struct {
		name string
		loc  diag.Location
		want string
	}{
		{"empty", diag.Location{}, ""},
		{"file only", diag.Location{File: "tapir.json"}, "tapir.json"},
		{"pointer only", diag.Location{Pointer: "/scenarios/login"}, "/scenarios/login"},
		{"file and pointer", diag.At("tapir.json", "/scenarios/login"), "tapir.json#/scenarios/login"},
		{"line", diag.Location{File: "tapir.json", Line: 12}, "tapir.json:12"},
		{"line and column", diag.Location{File: "tapir.json", Line: 12, Column: 7}, "tapir.json:12:7"},
		{"line wins over pointer", diag.Location{File: "tapir.json", Pointer: "/a", Line: 3, Column: 1}, "tapir.json:3:1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.loc.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
