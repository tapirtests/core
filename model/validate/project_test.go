package validate_test

import (
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

func TestProject(t *testing.T) {
	runBreakCases(t, []breakCase{
		// FormatVersion.
		{
			name:   "missing version",
			breaks: func(p *model.Project) { p.FormatVersion = "" },
			want:   []want{{validate.V0000, diag.Error, "/formatVersion"}},
		},
		{
			name:   "unsupported version",
			breaks: func(p *model.Project) { p.FormatVersion = "2" },
			want:   []want{{validate.V0001, diag.Error, "/formatVersion"}},
		},
		{
			name:   "version with spaces is not trimmed",
			breaks: func(p *model.Project) { p.FormatVersion = " 1" },
			want:   []want{{validate.V0001, diag.Error, "/formatVersion"}},
		},

		// Name.
		{
			name:   "missing name is a warning",
			breaks: func(p *model.Project) { p.Name = "" },
			want:   []want{{validate.V0100, diag.Warning, "/name"}},
		},

		// BaseURL.
		{
			name:   "missing base URL",
			breaks: func(p *model.Project) { p.BaseURL = "" },
			want:   []want{{validate.V0200, diag.Error, "/baseUrl"}},
		},

		// Spec.
		{
			name:   "no spec at all is valid",
			breaks: func(p *model.Project) { p.Spec = model.SpecRef{} },
			want:   nil,
		},
		{
			name:   "unsupported spec type",
			breaks: func(p *model.Project) { p.Spec.Type = "openapi3" },
			want:   []want{{validate.V0304, diag.Error, "/spec/type"}},
		},
		{
			name:   "missing spec type",
			breaks: func(p *model.Project) { p.Spec.Type = "" },
			want:   []want{{validate.V0301, diag.Error, "/spec/type"}},
		},
		{
			name:   "missing spec path",
			breaks: func(p *model.Project) { p.Spec.Path = "" },
			want:   []want{{validate.V0302, diag.Error, "/spec/path"}},
		},
		{
			name:   "missing spec hash",
			breaks: func(p *model.Project) { p.Spec.Hash = "" },
			want:   []want{{validate.V0303, diag.Error, "/spec/hash"}},
		},
		{
			// Every problem must be reported at once, not only the first one.
			name: "several spec fields missing are all reported",
			breaks: func(p *model.Project) {
				p.Spec.Type = ""
				p.Spec.Hash = ""
			},
			want: []want{
				{validate.V0301, diag.Error, "/spec/type"},
				{validate.V0303, diag.Error, "/spec/hash"},
			},
		},

		// Root.
		{
			name:   "missing root group",
			breaks: func(p *model.Project) { p.Root = nil },
			want:   []want{{validate.V0800, diag.Error, "/root"}},
		},
		{
			name:   "root group with a wrong name",
			breaks: func(p *model.Project) { p.Root.Name = "main" },
			want:   []want{{validate.V0801, diag.Error, "/root/name"}},
		},
		{
			name:   "root group without a name",
			breaks: func(p *model.Project) { p.Root.Name = "" },
			want:   []want{{validate.V0801, diag.Error, "/root/name"}},
		},

		// Independent problems in different fields are all reported.
		{
			name: "problems in several fields",
			breaks: func(p *model.Project) {
				p.FormatVersion = "2"
				p.Name = ""
				p.BaseURL = ""
			},
			want: []want{
				{validate.V0001, diag.Error, "/formatVersion"},
				{validate.V0100, diag.Warning, "/name"},
				{validate.V0200, diag.Error, "/baseUrl"},
			},
		},
	})
}
