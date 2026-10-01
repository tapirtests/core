package validate_test

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/internal/testutil"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

const testFile = "tapir.json"

// want describes an expected diagnostic. Messages are deliberately not
// compared: they may be reworded freely, codes and locations may not.
type want struct {
	code diag.Code
	sev  diag.Severity
	ptr  diag.Pointer
}

func (w want) String() string {
	return fmt.Sprintf("%s %s at %q", w.sev, w.code, w.ptr)
}

// breakCase is a "break one thing" test: it takes the valid project, breaks
// exactly one thing and lists the diagnostics that must appear — no more,
// no less.
type breakCase struct {
	name   string
	breaks func(p *model.Project)
	want   []want
}

// runBreakCases runs the cases as subtests.
func runBreakCases(t *testing.T, cases []breakCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testutil.ValidProject()
			tc.breaks(p)
			checkDiags(t, validate.Project(p, testFile), tc.want)
		})
	}
}

// checkDiags compares diagnostics with the expectation, ignoring order and
// messages.
func checkDiags(t *testing.T, got diag.List, wants []want) {
	t.Helper()

	gotW := make([]want, 0, len(got))
	for _, d := range got {
		if d.Location.File != testFile {
			t.Errorf("%s %s: file = %q, want %q", d.Severity, d.Code, d.Location.File, testFile)
		}
		gotW = append(gotW, want{code: d.Code, sev: d.Severity, ptr: d.Location.Pointer})
	}
	sortWants(gotW)
	wants = slices.Clone(wants)
	sortWants(wants)

	if !slices.Equal(gotW, wants) {
		t.Errorf("diagnostics mismatch\n got:\n%s\n want:\n%s\n full output:\n%s",
			listWants(gotW), listWants(wants), listDiags(got))
	}
}

func sortWants(ws []want) {
	slices.SortFunc(ws, func(a, b want) int {
		return cmp.Or(cmp.Compare(a.ptr, b.ptr), cmp.Compare(a.code, b.code), cmp.Compare(a.sev, b.sev))
	})
}

func listWants(ws []want) string {
	if len(ws) == 0 {
		return "  (none)"
	}
	var b strings.Builder
	for _, w := range ws {
		b.WriteString("  " + w.String() + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func listDiags(ds diag.List) string {
	if len(ds) == 0 {
		return "  (none)"
	}
	var b strings.Builder
	for _, d := range ds {
		b.WriteString("  " + d.String() + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// The reference project must be valid: every break case relies on it, so a
// diagnostic here would leak into all of them.
func TestValidProjectHasNoDiagnostics(t *testing.T) {
	checkDiags(t, validate.Project(testutil.ValidProject(), testFile), nil)
}

// The validator is called on whatever the loader managed to build, so it
// must survive incomplete projects.
func TestIncompleteProjectsDoNotPanic(t *testing.T) {
	cases := map[string]*model.Project{
		"nil project":   nil,
		"empty project": {},
		"empty root":    {FormatVersion: "1", Root: &model.Group{Name: "root"}},
	}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			validate.Project(p, testFile)
		})
	}
}
