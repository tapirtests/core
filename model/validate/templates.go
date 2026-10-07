package validate

import (
	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/random"
	"github.com/tapirtests/core/vars"
)

// validateTemplates checks the "{{...}}" templates inside a value written in
// the project: an input, a default, an expected value. It walks objects and
// arrays, so templates deep inside a request body are checked too; ptr points
// to the value, and diagnostics point to the exact string inside it.
//
// What can be checked before a run is checked here: the syntax, calls of
// random.* functions (their arguments are literals) and references to env
// variables (they are declared in the project). Whether a plain variable
// exists depends on what runs before the template is used and is not checked
// here.
func (v *validator) validateTemplates(value model.Value, ptr diag.Pointer) {
	for _, occ := range vars.Scan(value) {
		at := ptr
		for _, seg := range occ.Path {
			if seg.IsIndex {
				at = at.Index(seg.Index)
			} else {
				at = at.Key(seg.Field)
			}
		}

		if occ.Err != nil {
			v.errorf(V0900, at, "template %q: %s at offset %d", occ.Src, occ.Err.Msg, occ.Err.Pos)
			continue
		}

		for _, e := range occ.Template.Exprs() {
			switch e.Kind {
			case vars.ExprCall:
				if err := random.Check(e.Name, e.Args); err != nil {
					v.errorf(V0901, at, "{{%s}}: %v", e, err)
				}
			case vars.ExprEnv:
				if !v.envDeclared(e.Name) {
					v.errorf(V0902, at, "{{%s}}: env variable %q is not declared in the project", e, e.Name)
				}
			}
		}
	}
}

// envDeclared reports whether the project declares the env variable.
func (v *validator) envDeclared(name string) bool {
	if v.envNames == nil {
		v.envNames = make(map[string]bool, len(v.p.Env))
		for _, e := range v.p.Env {
			v.envNames[e.Name] = true
		}
	}
	return v.envNames[name]
}

// validateNotReserved reports a variable name that starts a namespace of the
// template language: a variable named "env" or "random" could never be read,
// because {{env.x}} and {{random.x}} mean something else. what names the
// declared thing in the message.
func (v *validator) validateNotReserved(name string, ptr diag.Pointer, what string) {
	if vars.IsReserved(name) {
		v.errorf(V0903, ptr, "%s name %q is reserved by the template language", what, name)
	}
}
