package validate

import "github.com/tapirtests/core/diag"

// Codes reported by the validator.
//
// Format: V<section><ordinal>, two digits each, so every code has the same
// length and codes sort in their natural order. Sections:
//
//	V00xx  formatVersion
//	V01xx  name
//	V02xx  baseUrl
//	V03xx  spec
//	V04xx  security schemes
//	V05xx  env
//	V06xx  requests
//	V07xx  scenarios
//	V08xx  groups
//
// A code is published once and never reused or renumbered: UIs, docs and
// user filters depend on it. Other components use their own letters:
// F (format), S (specification), N (env values).
const (
	// FormatVersion
	V0000 diag.Code = "V0000" // formatVersion is empty
	V0001 diag.Code = "V0001" // formatVersion is not supported by this core

	// Name
	V0100 diag.Code = "V0100" // project name is empty (warning)

	// BaseURL
	V0200 diag.Code = "V0200" // baseUrl is empty

	// Spec. The spec is optional (a project may consist of custom requests
	// only), but if any of its fields is set, all of them must be set.
	V0301 diag.Code = "V0301" // spec.type is empty while other spec fields are set
	V0302 diag.Code = "V0302" // spec.path is empty while other spec fields are set
	V0303 diag.Code = "V0303" // spec.hash is empty while other spec fields are set
	V0304 diag.Code = "V0304" // spec.type is not supported

	// SecuritySchemes
	V0400 diag.Code = "V0400" // security schemes are missed

	// Env

	// Requests

	// Scenarios

	// Root
	V0800 diag.Code = "V0800" // project has no root group
	V0801 diag.Code = "V0801" // root group is not named "root"
)
