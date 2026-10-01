package validate

import "github.com/tapirtests/core/diag"

const AvailableVersions = "1"

// Валидация глобальной части проекта (версия, env, security schemes)
func (v *validator) validateProject() {
	ptr := diag.Root.Key("tapirVersion")

	// Version
	switch {
	case len(v.p.TapirVersion) == 0:
		v.errorf(V000, ptr, "version is required")
	case v.p.TapirVersion != AvailableVersions:
		v.errorf(V001, ptr, "incorrect version")
	}

	// Name
	switch {
	case len(v.p.Name) == 0:
		v.warnf(V010, ptr, "name is missed")
	}

	// BaseURL
	switch {
	case len(v.p.BaseURL) == 0:
		v.errorf(V020, ptr, "base URL is required")
	}

	// Spec
	switch {
	case len(v.p.Spec.Type) == 0:
		v.errorf(V031, ptr, "spec type is missed")
	case len(v.p.Spec.Path) == 0:
		v.errorf(V032, ptr, "spec path is missed")
	case len(v.p.Spec.Hash) == 0:
		v.errorf(V033, ptr, "spec hash is missed")
	}

}
