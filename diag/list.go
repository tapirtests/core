package diag

import (
	"cmp"
	"slices"
)

// List is a collection of diagnostics, usually everything found by a single
// check (load, validate, parse).
type List []Diagnostic

// Add appends diagnostics to the list.
//
//goland:noinspection GoMixedReceiverTypes
func (l *List) Add(ds ...Diagnostic) {
	*l = append(*l, ds...)
}

// HasErrors reports whether the list contains at least one Error.
// A run must not start if it does.
//
//goland:noinspection GoMixedReceiverTypes
func (l List) HasErrors() bool {
	return slices.ContainsFunc(l, func(d Diagnostic) bool {
		return d.Severity == Error
	})
}

// Count returns the number of diagnostics with the given severity.
//
//goland:noinspection GoMixedReceiverTypes
func (l List) Count(sev Severity) int {
	n := 0
	for _, d := range l {
		if d.Severity == sev {
			n++
		}
	}
	return n
}

// Sort orders diagnostics by file, then by position in the file, then by
// severity (errors first). The sort is stable, so diagnostics at the same
// place keep the order in which they were reported.
func (l List) Sort() {
	slices.SortStableFunc(l, func(a, b Diagnostic) int {
		return cmp.Or(
			cmp.Compare(a.Location.File, b.Location.File),
			compareLines(a.Location, b.Location),
			cmp.Compare(a.Location.Pointer, b.Location.Pointer),
			cmp.Compare(a.Severity, b.Severity),
		)
	})
}

// compareLines orders by line and column; locations without a line go last,
// since they cannot be placed relative to the ones that have it.
func compareLines(a, b Location) int {
	switch {
	case a.Line == 0 && b.Line == 0:
		return 0
	case a.Line == 0:
		return 1
	case b.Line == 0:
		return -1
	}
	return cmp.Or(cmp.Compare(a.Line, b.Line), cmp.Compare(a.Column, b.Column))
}
