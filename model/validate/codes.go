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

	// Scenarios: the definition itself.
	V0700 diag.Code = "V0700" // scenario entry is nil
	V0701 diag.Code = "V0701" // scenario ID is empty
	V0702 diag.Code = "V0702" // scenario ID differs from its key in scenarios

	// Scenarios: signature.
	V0703 diag.Code = "V0703" // scenario input name is empty
	V0704 diag.Code = "V0704" // scenario input name is not an identifier
	V0705 diag.Code = "V0705" // scenario input is declared more than once
	V0706 diag.Code = "V0706" // scenario output name is empty
	V0707 diag.Code = "V0707" // scenario output name is not an identifier
	V0708 diag.Code = "V0708" // scenario output is declared more than once
	V0709 diag.Code = "V0709" // scenario output has no value
	V0710 diag.Code = "V0710" // required scenario input has a default that is never used (warning)

	// Scenarios: steps.
	V0711 diag.Code = "V0711" // scenario has no steps (warning)
	V0712 diag.Code = "V0712" // step is nil
	V0713 diag.Code = "V0713" // step ID is empty
	V0714 diag.Code = "V0714" // step ID is used more than once in the scenario
	V0715 diag.Code = "V0715" // step kind is not supported

	// Scenarios: request call steps.
	V0716 diag.Code = "V0716" // request call has no request ID
	V0717 diag.Code = "V0717" // request call refers to an unknown request
	V0718 diag.Code = "V0718" // request call passes an input the request does not declare
	V0719 diag.Code = "V0719" // request call does not pass a required input
	V0720 diag.Code = "V0720" // request call timeout is negative
	V0721 diag.Code = "V0721" // extracted variable name is empty
	V0722 diag.Code = "V0722" // extracted variable name is not an identifier
	V0723 diag.Code = "V0723" // extract path does not start with "$"

	// Scenarios: assertions.
	V0724 diag.Code = "V0724" // assertion target is empty
	V0725 diag.Code = "V0725" // assertion target is not supported
	V0726 diag.Code = "V0726" // assertion operator is empty
	V0727 diag.Code = "V0727" // assertion operator is not supported
	V0728 diag.Code = "V0728" // assertion operator is not applicable to its target
	V0729 diag.Code = "V0729" // assertion path must be empty for status and duration
	V0730 diag.Code = "V0730" // assertion path is required for body and header
	V0731 diag.Code = "V0731" // body assertion path does not start with "$"
	V0732 diag.Code = "V0732" // assertion operator requires a value
	V0733 diag.Code = "V0733" // assertion value is ignored by exists/notExists (warning)
	V0734 diag.Code = "V0734" // "in" requires an array value
	V0735 diag.Code = "V0735" // "matches" requires a valid regular expression
	V0736 diag.Code = "V0736" // comparison and "length" require a number

	// Root
	V0800 diag.Code = "V0800" // project has no root group
	V0801 diag.Code = "V0801" // root group is not named "root"
)
