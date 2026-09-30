package validate

import "github.com/tapirtests/core/diag"

// Валидация глобальной части проекта (версия, env, security schemes)
func (v *validator) validateProject() {
	ptr := diag.Root.Key("tapirVersion")

	switch {
	case len(v.p.TapirVersion) == 0:
		v.warnf(V000, ptr, "tapir version is required")
	case v.p.TapirVersion != "1":
		v.warnf(V001, ptr, "incorrect tapir version")
	}

}
