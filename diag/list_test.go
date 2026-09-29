package diag_test

import (
	"testing"

	"github.com/tapirtests/core/diag"
)

func TestListCounts(t *testing.T) {
	var l diag.List
	if l.HasErrors() {
		t.Error("empty list has errors")
	}

	l.Add(
		diag.Warningf("W1", diag.Location{}, "w"),
		diag.Infof("I1", diag.Location{}, "i"),
	)
	if l.HasErrors() {
		t.Error("list without errors reports errors")
	}

	l.Add(diag.Errorf("E1", diag.Location{}, "e"), diag.Warningf("W2", diag.Location{}, "w"))
	if !l.HasErrors() {
		t.Error("list with an error reports no errors")
	}

	for sev, want := range map[diag.Severity]int{diag.Error: 1, diag.Warning: 2, diag.Info: 1} {
		if got := l.Count(sev); got != want {
			t.Errorf("Count(%v) = %d, want %d", sev, got, want)
		}
	}
}

func TestListSort(t *testing.T) {
	l := diag.List{
		diag.Infof("no-line", diag.At("tapir.json", "/a"), ""),
		diag.Warningf("line-10", diag.Location{File: "tapir.json", Line: 10}, ""),
		diag.Errorf("spec", diag.Location{File: "petstore.yaml", Line: 99}, ""),
		diag.Errorf("line-2-col-5", diag.Location{File: "tapir.json", Line: 2, Column: 5}, ""),
		diag.Warningf("line-2-col-1-warning", diag.Location{File: "tapir.json", Line: 2, Column: 1}, ""),
		diag.Errorf("line-2-col-1-error", diag.Location{File: "tapir.json", Line: 2, Column: 1}, ""),
		diag.Warningf("same-place-second", diag.Location{File: "tapir.json", Line: 10}, ""),
	}
	l.Sort()

	want := []diag.Code{
		"spec",                 // "petstore.yaml" < "tapir.json"
		"line-2-col-1-error",   // same position: errors first
		"line-2-col-1-warning", //
		"line-2-col-5",         //
		"line-10",              // stable: reported before "same-place-second"
		"same-place-second",    //
		"no-line",              // locations without a line go last
	}
	if len(l) != len(want) {
		t.Fatalf("len = %d, want %d", len(l), len(want))
	}
	for i, d := range l {
		if d.Code != want[i] {
			t.Errorf("position %d: got %q, want %q", i, d.Code, want[i])
		}
	}
}
