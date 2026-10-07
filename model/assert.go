package model

// Assertion is a single check of a response, written in a RequestCall:
// "$.status equals 201", "$.body.name equals Phone", "$.duration lt 500".
//
// What is checked is selected by Path, an exact JSONPath from the root of the
// response document, the same document extraction works on:
//
//	$.status                     the status of the response
//	$.headers['content-type']    a response header (names are lower-case)
//	$.body.items[0].name         a value in the response body
//	$.duration                   the duration of the call in milliseconds
//
// The expected result is part of a usage, not of a definition: the same
// request may be expected to return 200 in one scenario and 404 in another.
type Assertion struct {
	Path  string   // exact JSONPath from the root of the response document
	Op    AssertOp // comparison operator: equals, notEquals, in, exists, contains, matches, ...
	Value Value    // expected value; may contain templates, e.g. "{{petName}}"; unused by exists/notExists
}

// AssertOp is the comparison operator of an Assertion.
type AssertOp string

const (
	// OpEquals - the actual value deeply equals Value.
	OpEquals AssertOp = "equals"
	// OpNotEquals - the actual value does not equal Value.
	OpNotEquals AssertOp = "notEquals"
	// OpIn - the actual value equals one of the elements of Value, which must be an array.
	OpIn AssertOp = "in"
	// OpExists - the value selected by Path is present (it may be null). Value is unused.
	OpExists AssertOp = "exists"
	// OpNotExists - the value selected by Path is absent. Value is unused.
	OpNotExists AssertOp = "notExists"
	// OpContains - the actual string contains the Value substring,
	// or the actual array contains an element equal to Value.
	OpContains AssertOp = "contains"
	// OpMatches - the actual string matches the regular expression in Value.
	OpMatches AssertOp = "matches"
	// OpLt - the actual number is less than Value.
	OpLt AssertOp = "lt"
	// OpLte - the actual number is less than or equal to Value.
	OpLte AssertOp = "lte"
	// OpGt - the actual number is greater than Value.
	OpGt AssertOp = "gt"
	// OpGte - the actual number is greater than or equal to Value.
	OpGte AssertOp = "gte"
	// OpLength - the length of the actual string, array or object equals Value.
	OpLength AssertOp = "length"
)
