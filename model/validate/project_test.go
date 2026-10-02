package validate_test

import (
	"testing"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// A fresh project has no requests, scenarios, env variables or security
// schemes yet; empty buckets must not be reported.
func TestMinimalProjectIsValid(t *testing.T) {
	p := &model.Project{
		FormatVersion: "1",
		Name:          "empty",
		BaseURL:       "http://localhost:8080",
		Root:          &model.Group{Name: "root"},
	}
	checkDiags(t, validate.Project(p, testFile), nil)
}

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

		// Security schemes.
		{
			name: "custom scheme name is valid",
			breaks: func(p *model.Project) {
				p.SecuritySchemes["serviceKey"] = &model.SecurityScheme{
					Name: "serviceKey", Type: model.SecurityAPIKey, In: model.InQuery, ParamName: "api_key",
				}
			},
			want: nil,
		},
		{
			name:   "nil security scheme",
			breaks: func(p *model.Project) { p.SecuritySchemes["broken"] = nil },
			want:   []want{{validate.V0400, diag.Error, "/securitySchemes/broken"}},
		},
		{
			name:   "missing security scheme name",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].Name = "" },
			want:   []want{{validate.V0401, diag.Error, "/securitySchemes/bearer/name"}},
		},
		{
			name:   "security scheme name differs from key",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].Name = "jwt" },
			want:   []want{{validate.V0402, diag.Error, "/securitySchemes/bearer/name"}},
		},
		{
			name:   "missing security scheme type",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].Type = "" },
			want:   []want{{validate.V0403, diag.Error, "/securitySchemes/bearer/type"}},
		},
		{
			name:   "unsupported security scheme type",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].Type = "basic" },
			want:   []want{{validate.V0404, diag.Error, "/securitySchemes/bearer/type"}},
		},
		{
			name:   "missing security scheme location",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].In = "" },
			want:   []want{{validate.V0405, diag.Error, "/securitySchemes/bearer/in"}},
		},
		{
			name:   "unsupported security scheme location",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].In = "cookie" },
			want:   []want{{validate.V0406, diag.Error, "/securitySchemes/bearer/in"}},
		},
		{
			name:   "missing security scheme parameter name",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].ParamName = "" },
			want:   []want{{validate.V0407, diag.Error, "/securitySchemes/bearer/paramName"}},
		},
		{
			name:   "empty prefix is valid",
			breaks: func(p *model.Project) { p.SecuritySchemes["bearer"].Prefix = "" },
			want:   nil,
		},
		{
			name: "key with a slash is escaped in the pointer",
			breaks: func(p *model.Project) {
				p.SecuritySchemes["a/b"] = &model.SecurityScheme{
					Name: "a/b", Type: model.SecurityAPIKey, In: model.InHeader,
				}
			},
			want: []want{{validate.V0407, diag.Error, "/securitySchemes/a~1b/paramName"}},
		},
		{
			name: "several problems in one scheme are all reported",
			breaks: func(p *model.Project) {
				s := p.SecuritySchemes["bearer"]
				s.Type = ""
				s.In = "cookie"
				s.ParamName = ""
			},
			want: []want{
				{validate.V0403, diag.Error, "/securitySchemes/bearer/type"},
				{validate.V0406, diag.Error, "/securitySchemes/bearer/in"},
				{validate.V0407, diag.Error, "/securitySchemes/bearer/paramName"},
			},
		},

		// Env.
		{
			name:   "missing env name",
			breaks: func(p *model.Project) { p.Env[0].Name = "" },
			want:   []want{{validate.V0500, diag.Error, "/env/0/name"}},
		},
		{
			name:   "env name with a dash",
			breaks: func(p *model.Project) { p.Env[1].Name = "SELLER-USER" },
			want:   []want{{validate.V0501, diag.Error, "/env/1/name"}},
		},
		{
			name:   "env name starting with a digit",
			breaks: func(p *model.Project) { p.Env[1].Name = "1SELLER" },
			want:   []want{{validate.V0501, diag.Error, "/env/1/name"}},
		},
		{
			name:   "lowercase env name is valid",
			breaks: func(p *model.Project) { p.Env[1].Name = "seller_user" },
			want:   nil,
		},
		{
			name: "duplicate env name is reported on the repeat only",
			breaks: func(p *model.Project) {
				p.Env = append(p.Env, model.EnvVarDecl{Name: "BASE_URL"})
			},
			want: []want{{validate.V0502, diag.Error, "/env/5/name"}},
		},
		{
			name: "every repeat of an env name is reported",
			breaks: func(p *model.Project) {
				p.Env = append(p.Env, model.EnvVarDecl{Name: "BASE_URL"}, model.EnvVarDecl{Name: "BASE_URL"})
			},
			want: []want{
				{validate.V0502, diag.Error, "/env/5/name"},
				{validate.V0502, diag.Error, "/env/6/name"},
			},
		},
		{
			name: "empty env names are not reported as duplicates",
			breaks: func(p *model.Project) {
				p.Env[0].Name = ""
				p.Env[1].Name = ""
			},
			want: []want{
				{validate.V0500, diag.Error, "/env/0/name"},
				{validate.V0500, diag.Error, "/env/1/name"},
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
