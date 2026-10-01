package validate

import (
	"slices"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// supportedVersion is the only tapir.json format version this core reads.
const supportedVersion = "1"

// rootGroupName is the required name of Project.Root.
const rootGroupName = "root"

// supportedSpecTypes lists the specification formats the core can import.
var supportedSpecTypes = []string{"swagger2"}

// validateProject checks the top-level fields of the project: format version,
// name, base URL, specification reference and the presence of the root group.
func (v *validator) validateProject() {
	v.validateVersion()

	if v.p.Name == "" {
		v.warnf(V010, diag.Root.Key("name"), "project name is empty")
	}

	if v.p.BaseURL == "" {
		v.errorf(V020, diag.Root.Key("baseUrl"), "base URL is required")
	}

	v.validateSpec()
	v.validateRoot()
}

func (v *validator) validateVersion() {
	ptr := diag.Root.Key("tapirVersion")
	switch v.p.TapirVersion {
	case supportedVersion:
	case "":
		v.errorf(V000, ptr, "tapir version is required")
	default:
		v.errorf(V001, ptr, "unsupported tapir version %q, supported: %q",
			v.p.TapirVersion, supportedVersion)
	}
}

// validateSpec checks the specification reference. A project without a
// specification is valid (custom requests only), so an entirely empty
// reference is accepted; a partially filled one is not.
func (v *validator) validateSpec() {
	spec := v.p.Spec
	if spec == (model.SpecRef{}) {
		return
	}

	ptr := diag.Root.Key("spec")
	if spec.Type == "" {
		v.errorf(V031, ptr.Key("type"), "spec type is required when a spec is set")
	} else if !slices.Contains(supportedSpecTypes, spec.Type) {
		v.errorf(V034, ptr.Key("type"), "unsupported spec type %q, supported: %q",
			spec.Type, supportedSpecTypes)
	}
	if spec.Path == "" {
		v.errorf(V032, ptr.Key("path"), "spec path is required when a spec is set")
	}
	if spec.Hash == "" {
		v.errorf(V033, ptr.Key("hash"), "spec hash is required when a spec is set")
	}
}

func (v *validator) validateRoot() {
	ptr := diag.Root.Key("root")
	if v.p.Root == nil {
		v.errorf(V080, ptr, "root group is required")
		return
	}
	if v.p.Root.Name != rootGroupName {
		v.errorf(V081, ptr.Key("name"), "root group must be named %q, got %q",
			rootGroupName, v.p.Root.Name)
	}
}
