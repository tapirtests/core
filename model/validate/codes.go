package validate

import "github.com/tapirtests/core/diag"

// F — format
// V — validate
// S — спецификация
// N — env

const (
	// TapirVersion
	V000 diag.Code = "V000" // Missing TapirVersion field
	V001 diag.Code = "V001" // Unsupported TapirVersion field

	// Name
	V010 diag.Code = "V010" // Missing Name field

	// BaseURL
	V020 diag.Code = "V020" // Missing BaseURL field

	// Spec
	V030 diag.Code = "V030" // Missing Spec field
	V031 diag.Code = "V031" // Missing Spec name field
	V032 diag.Code = "V032" // Missing Spec path field
	V033 diag.Code = "V033" // Missing Spec hash field

	// SecuritySchemes
	V040 diag.Code = "V040" // Missing SecuritySchemes field

	// Env
	V050 diag.Code = "V050" // Missing Env field

	// Requests
	V060 diag.Code = "V060" // Missing Requests field

	// Scenarios
	V070 diag.Code = "V070" // Missing Scenarios field

	// Root
	V080 diag.Code = "V080" // Missing Root field
)
