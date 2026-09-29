package diag_test

import (
	"encoding/json"
	"testing"

	"github.com/tapirtests/core/diag"
)

func TestConstructors(t *testing.T) {
	loc := diag.At("tapir.json", "/scenarios/login")
	tests := []struct {
		name string
		d    diag.Diagnostic
		want diag.Severity
	}{
		{"error", diag.Errorf("TAPIR-E001", loc, "bad %s", "input"), diag.Error},
		{"warning", diag.Warningf("TAPIR-W001", loc, "bad %s", "input"), diag.Warning},
		{"info", diag.Infof("TAPIR-I001", loc, "bad %s", "input"), diag.Info},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.d.Severity != tt.want {
				t.Errorf("severity = %v, want %v", tt.d.Severity, tt.want)
			}
			if tt.d.Message != "bad input" {
				t.Errorf("message = %q, want %q", tt.d.Message, "bad input")
			}
			if tt.d.Location != loc {
				t.Errorf("location = %+v, want %+v", tt.d.Location, loc)
			}
		})
	}
}

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		name string
		d    diag.Diagnostic
		want string
	}{
		{
			name: "full",
			d:    diag.Errorf("TAPIR-E012", diag.Location{File: "tapir.json", Line: 148, Column: 17}, `unknown input "age"`),
			want: `tapir.json:148:17: error TAPIR-E012: unknown input "age"`,
		},
		{
			name: "no location",
			d:    diag.Warningf("TAPIR-W004", diag.Location{}, "something"),
			want: "warning TAPIR-W004: something",
		},
		{
			name: "no code",
			d:    diag.Infof("", diag.Location{File: "petstore.yaml"}, "12 operations have no operationId"),
			want: "petstore.yaml: info: 12 operations have no operationId",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSeverityString(t *testing.T) {
	if got := diag.Severity(42).String(); got != "Severity(42)" {
		t.Errorf("unknown severity string = %q", got)
	}
}

func TestDiagnosticJSON(t *testing.T) {
	d := diag.Errorf("TAPIR-E012", diag.Location{File: "tapir.json", Pointer: "/scenarios/a", Line: 3}, "msg")

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"severity":"error","code":"TAPIR-E012","location":{"file":"tapir.json","pointer":"/scenarios/a","line":3},"message":"msg"}`
	if string(data) != want {
		t.Errorf("json:\n got  %s\n want %s", data, want)
	}

	var back diag.Diagnostic
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back != d {
		t.Errorf("round trip: got %+v, want %+v", back, d)
	}
}

func TestSeverityJSONErrors(t *testing.T) {
	if _, err := json.Marshal(diag.Severity(0)); err == nil {
		t.Error("marshal of zero severity: expected error")
	}
	var s diag.Severity
	if err := json.Unmarshal([]byte(`"fatal"`), &s); err == nil {
		t.Error("unmarshal of unknown severity: expected error")
	}
}
