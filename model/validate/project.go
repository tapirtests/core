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

// supportedSecurityTypes and supportedSecurityIn list the kinds of security
// schemes the core can apply to a request.
var (
	supportedSecurityTypes = []model.SecurityType{model.SecurityAPIKey}
	supportedSecurityIn    = []model.ParamIn{model.InHeader, model.InQuery}
)

// validateProject checks the global part of the project: format version,
// name, base URL, specification reference, security schemes, env variables
// and the presence of the root group.
func (v *validator) validateProject() {
	v.validateVersion()

	if v.p.Name == "" {
		v.warnf(V0100, diag.Root.Key("name"), "project name is empty")
	}

	if v.p.BaseURL == "" {
		v.errorf(V0200, diag.Root.Key("baseUrl"), "base URL is required")
	}

	v.validateSpec()
	v.validateSecuritySchemes()
	v.validateEnv()
	v.validateRoot()
}

func (v *validator) validateVersion() {
	ptr := diag.Root.Key("formatVersion")
	switch v.p.FormatVersion {
	case supportedVersion:
	case "":
		v.errorf(V0000, ptr, "format version is required")
	default:
		v.errorf(V0001, ptr, "unsupported format version %q, supported: %q",
			v.p.FormatVersion, supportedVersion)
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
		v.errorf(V0301, ptr.Key("type"), "spec type is required when a spec is set")
	} else if !slices.Contains(supportedSpecTypes, spec.Type) {
		v.errorf(V0304, ptr.Key("type"), "unsupported spec type %q, supported: %q",
			spec.Type, supportedSpecTypes)
	}
	if spec.Path == "" {
		v.errorf(V0302, ptr.Key("path"), "spec path is required when a spec is set")
	}
	if spec.Hash == "" {
		v.errorf(V0303, ptr.Key("hash"), "spec hash is required when a spec is set")
	}
}

// validateSecuritySchemes checks every security scheme. Having no schemes is
// valid: an API may require no authorization at all. Scheme names are free:
// they come from the specification; only the kind of scheme is limited.
func (v *validator) validateSecuritySchemes() {
	base := diag.Root.Key("securitySchemes")
	for key, scheme := range v.p.SecuritySchemes {
		ptr := base.Key(key)
		if scheme == nil {
			v.errorf(V0400, ptr, "security scheme %q is empty", key)
			continue
		}

		switch scheme.Name {
		case key:
		case "":
			v.errorf(V0401, ptr.Key("name"), "security scheme name is required")
		default:
			v.errorf(V0402, ptr.Key("name"), "security scheme name %q differs from its key %q",
				scheme.Name, key)
		}

		switch {
		case scheme.Type == "":
			v.errorf(V0403, ptr.Key("type"), "security scheme type is required")
		case !slices.Contains(supportedSecurityTypes, scheme.Type):
			v.errorf(V0404, ptr.Key("type"), "unsupported security scheme type %q, supported: %q",
				scheme.Type, supportedSecurityTypes)
		}

		switch {
		case scheme.In == "":
			v.errorf(V0405, ptr.Key("in"), "security scheme location (in) is required")
		case !slices.Contains(supportedSecurityIn, scheme.In):
			v.errorf(V0406, ptr.Key("in"), "unsupported security scheme location %q, supported: %q",
				scheme.In, supportedSecurityIn)
		}

		if scheme.ParamName == "" {
			v.errorf(V0407, ptr.Key("paramName"), "security scheme parameter name is required")
		}
	}
}

// validateEnv checks env variable declarations. Duplicates are reported on
// every repeated declaration, the first one is considered the original.
func (v *validator) validateEnv() {
	base := diag.Root.Key("env")
	seen := make(map[string]bool, len(v.p.Env))
	for i, e := range v.p.Env {
		ptr := base.Index(i).Key("name")
		switch {
		case e.Name == "":
			v.errorf(V0500, ptr, "env variable name is required")
			continue
		case !identifierPattern.MatchString(e.Name):
			v.errorf(V0501, ptr, "env variable name %q must match %s", e.Name, identifierPattern)
		}

		if seen[e.Name] {
			v.errorf(V0502, ptr, "env variable %q is declared more than once", e.Name)
		}
		seen[e.Name] = true
	}
}

func (v *validator) validateRoot() {
	ptr := diag.Root.Key("root")
	if v.p.Root == nil {
		v.errorf(V0800, ptr, "root group is required")
		return
	}
	if v.p.Root.Name != rootGroupName {
		v.errorf(V0801, ptr.Key("name"), "root group must be named %q, got %q",
			rootGroupName, v.p.Root.Name)
	}
}
