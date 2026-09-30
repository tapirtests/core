package validate

import (
	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

func Project(p *model.Project, file string) diag.List {
	val := validator{p: p, file: file, diags: make([]diag.Diagnostic, 0)}

	val.validateProject()

	val.diags.Sort()
	return val.diags
}

type validator struct {
	p     *model.Project
	file  string
	diags diag.List
}

func (v *validator) errorf(code diag.Code, ptr diag.Pointer, format string, args ...any) {
	v.diags.Add(diag.Errorf(code, diag.Location{File: v.file, Pointer: ptr}, format, args...))
}

func (v *validator) warnf(code diag.Code, ptr diag.Pointer, format string, args ...any) {
	v.diags.Add(diag.Warningf(code, diag.Location{File: v.file, Pointer: ptr}, format, args...))
}
