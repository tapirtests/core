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

	// SecuritySchemes. Scheme names are chosen by the API author and may be
	// anything; what is limited is the kind of scheme the core can apply.
	V0400 diag.Code = "V0400" // security scheme entry is nil
	V0401 diag.Code = "V0401" // security scheme name is empty
	V0402 diag.Code = "V0402" // security scheme name differs from its key in securitySchemes
	V0403 diag.Code = "V0403" // security scheme type is empty
	V0404 diag.Code = "V0404" // security scheme type is not supported
	V0405 diag.Code = "V0405" // security scheme location (in) is empty
	V0406 diag.Code = "V0406" // security scheme location (in) is not supported
	V0407 diag.Code = "V0407" // security scheme parameter name is empty

	// Env
	V0500 diag.Code = "V0500" // env variable name is empty
	V0501 diag.Code = "V0501" // env variable name is not an identifier
	V0502 diag.Code = "V0502" // env variable is declared more than once

	// Requests: the definition itself.
	V0600 diag.Code = "V0600" // request entry is nil
	V0601 diag.Code = "V0601" // request ID is empty
	V0602 diag.Code = "V0602" // request ID differs from its key in requests
	V0603 diag.Code = "V0603" // request has no operation
	V0604 diag.Code = "V0604" // operation protocol is not supported

	// Requests: HTTP operation.
	V0605 diag.Code = "V0605" // HTTP method is empty
	V0606 diag.Code = "V0606" // HTTP method is not supported
	V0607 diag.Code = "V0607" // HTTP path is empty
	V0608 diag.Code = "V0608" // HTTP path does not start with "/"
	V0609 diag.Code = "V0609" // HTTP path template is malformed: unbalanced or empty {}
	V0610 diag.Code = "V0610" // path parameter {x} has no path.x input

	// Requests: input contract.
	V0611 diag.Code = "V0611" // input name is empty
	V0612 diag.Code = "V0612" // input location (name prefix) is unknown
	V0613 diag.Code = "V0613" // input name has no field after its location, e.g. "query."
	V0614 diag.Code = "V0614" // input is declared more than once
	V0615 diag.Code = "V0615" // path.x input has no {x} in the path template
	V0616 diag.Code = "V0616" // path input is not required
	V0617 diag.Code = "V0617" // auth input refers to an unknown security scheme
	V0618 diag.Code = "V0618" // whole-body input "body" is combined with body.<field> inputs

	// Requests: output contract and schemas.
	V0619 diag.Code = "V0619" // response key is neither an HTTP status code nor "default"
	V0620 diag.Code = "V0620" // schema is not valid JSON

	// Scenarios

	// Root
	V0800 diag.Code = "V0800" // project has no root group
	V0801 diag.Code = "V0801" // root group is not named "root"
)
