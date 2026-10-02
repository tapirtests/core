// Package validate checks a model.Project for semantic problems: broken
// references, contract violations, duplicates. It runs after the project was
// loaded and before it is executed.
package validate

import (
	"regexp"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// Project checks p and returns every problem found, sorted by location.
// file is the name of the document the project was loaded from; it is put
// into every diagnostic.
//
// Project never modifies p and never panics on incomplete projects (nil maps,
// nil root): the loader may hand over whatever it managed to build. A nil
// project yields no diagnostics.
func Project(p *model.Project, file string) diag.List {
	if p == nil {
		return nil
	}

	val := validator{p: p, file: file}

	val.validateProject()
	val.validateRequests()
	val.validateScenarios()
	val.validateGroups()

	val.diags.Sort()
	return val.diags
}

// validator holds the state of one Project check; each section of the
// project is checked by its own method.
type validator struct {
	p     *model.Project
	file  string
	diags diag.List
}

func (v *validator) errorf(code diag.Code, ptr diag.Pointer, format string, args ...any) {
	v.diags.Add(diag.Errorf(code, diag.At(v.file, ptr), format, args...))
}

func (v *validator) warnf(code diag.Code, ptr diag.Pointer, format string, args ...any) {
	v.diags.Add(diag.Warningf(code, diag.At(v.file, ptr), format, args...))
}

// identifierPattern is the allowed form of a declared name: env variables,
// scenario inputs and outputs, extracted variables. Such names are used in
// templates ({{name}}, {{env.NAME}}) and in .env files. Dots are not allowed:
// they are reserved for automatic output names "<alias>.<output>", so a
// declared name never clashes with one.
var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
