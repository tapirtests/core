package validate

import "github.com/tapirtests/core/diag"

// Codes reported by the validator.
//
// Format: V<section><ordinal>. Sections:
//
//	V00x  tapirVersion
//	V01x  name
//	V02x  baseUrl
//	V03x  spec
//	V04x  security schemes
//	V05x  env
//	V06x  requests
//	V07x  scenarios
//	V08x  groups
//
// A code is published once and never reused or renumbered: UIs, docs and
// user filters depend on it. Other components use their own letters:
// F (format), S (specification), N (env values).
const (
	// TapirVersion
	V000 diag.Code = "V000" // tapirVersion is empty
	V001 diag.Code = "V001" // tapirVersion is not supported by this core

	// Name
	V010 diag.Code = "V010" // project name is empty (warning)

	// BaseURL
	V020 diag.Code = "V020" // baseUrl is empty

	// Spec. The spec is optional (a project may consist of custom requests
	// only), but if any of its fields is set, all of them must be set.
	V031 diag.Code = "V031" // spec.type is empty while other spec fields are set
	V032 diag.Code = "V032" // spec.path is empty while other spec fields are set
	V033 diag.Code = "V033" // spec.hash is empty while other spec fields are set
	V034 diag.Code = "V034" // spec.type is not supported

	// SecuritySchemes

	// Env

	// Requests

	// Scenarios

	// Root
	V080 diag.Code = "V080" // project has no root group
	V081 diag.Code = "V081" // root group is not named "root"
)
