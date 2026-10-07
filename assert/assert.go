// Package assert evaluates assertions against a response.
//
// The validator checks that an assertion is well-formed before a run; this
// package computes it during a run: it takes the value the path points to in
// the response document and compares it with the expected value.
//
// There are two different negative outcomes:
//
//   - the assertion does not hold: the response is not what was expected.
//     Check returns a Result with Passed false and a message;
//   - the assertion cannot be evaluated: the assertion itself is wrong (a
//     regular expression that does not compile, an expected value of the
//     wrong type). Check returns an error. That is a problem of the test,
//     not of the API under test.
//
// An actual value of an unexpected type ("lt 10" while the response has a
// string) is the first kind: the API returned something else than expected.
package assert

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tapirtests/core/extract"
	"github.com/tapirtests/core/model"
)

// Result is the outcome of one assertion.
type Result struct {
	Passed bool
	// Path and Op repeat the assertion; Expected is its value, with templates
	// already resolved. Expected is nil for exists and notExists.
	Path     string
	Op       model.AssertOp
	Expected any
	// Actual is the value found at Path. Found tells whether there is one:
	// a null in the response is found, with Actual nil.
	Actual any
	Found  bool
	// Message explains a failed assertion; empty if it passed.
	Message string
}

// Check evaluates an assertion against a response document, as built by
// extract.Document. The Value of the assertion must be resolved: templates
// are evaluated by the caller, the engine.
//
// If the path points to nothing, notExists passes and every other operator
// fails, notEquals included: nothing can be said about a value that is not
// there.
func Check(a model.Assertion, doc any) (Result, error) {
	res := Result{Path: a.Path, Op: a.Op, Expected: a.Value}

	path, err := extract.Parse(a.Path)
	if err != nil {
		return res, fmt.Errorf("assertion path %q: %w", a.Path, err)
	}
	check, ok := checks[a.Op]
	if !ok {
		return res, fmt.Errorf("unknown assertion operator %q", a.Op)
	}

	res.Actual, res.Found = path.Get(doc)

	switch a.Op {
	case model.OpExists:
		res.Expected = nil
		res.Passed = res.Found
		if !res.Found {
			res.Message = fmt.Sprintf("expected %s to exist, but there is nothing at this path", a.Path)
		}
		return res, nil
	case model.OpNotExists:
		res.Expected = nil
		res.Passed = !res.Found
		if res.Found {
			res.Message = fmt.Sprintf("expected %s not to exist, got %s", a.Path, show(res.Actual))
		}
		return res, nil
	}

	// Check the expected value first: a malformed assertion is an error
	// whether or not the response has a value to compare with.
	problem, err := check(res.Actual, a.Value, !res.Found)
	if err != nil {
		return res, fmt.Errorf("assertion %s %s: %w", a.Path, a.Op, err)
	}
	if !res.Found {
		res.Message = fmt.Sprintf("expected %s %s, but there is nothing at this path", a.Path, describe(a.Op, a.Value))
		return res, nil
	}
	if problem != "" {
		res.Message = fmt.Sprintf("expected %s %s, %s", a.Path, describe(a.Op, a.Value), problem)
		return res, nil
	}
	res.Passed = true
	return res, nil
}

// checkFunc evaluates one operator. It returns an error if the expected
// value is unusable, and otherwise a description of why the actual value
// does not satisfy the operator, empty if it does. With validateOnly set
// there is no actual value: only the expected one is checked.
type checkFunc func(actual, expected any, validateOnly bool) (problem string, err error)

var checks = map[model.AssertOp]checkFunc{
	model.OpEquals:    checkEquals,
	model.OpNotEquals: checkNotEquals,
	model.OpIn:        checkIn,
	model.OpContains:  checkContains,
	model.OpMatches:   checkMatches,
	model.OpLt:        compare(func(c int) bool { return c < 0 }),
	model.OpLte:       compare(func(c int) bool { return c <= 0 }),
	model.OpGt:        compare(func(c int) bool { return c > 0 }),
	model.OpGte:       compare(func(c int) bool { return c >= 0 }),
	model.OpLength:    checkLength,
	// exists and notExists need no comparison and are handled in Check.
	model.OpExists:    nil,
	model.OpNotExists: nil,
}

func checkEquals(actual, expected any, validateOnly bool) (string, error) {
	if validateOnly || equal(actual, expected) {
		return "", nil
	}
	return "got " + show(actual) + typeHint(actual, expected), nil
}

func checkNotEquals(actual, expected any, validateOnly bool) (string, error) {
	if validateOnly || !equal(actual, expected) {
		return "", nil
	}
	return "but it is equal", nil
}

func checkIn(actual, expected any, validateOnly bool) (string, error) {
	options, ok := expected.([]any)
	if !ok {
		return "", fmt.Errorf("expected value must be an array, got %s", kindOf(expected))
	}
	if validateOnly {
		return "", nil
	}
	for _, option := range options {
		if equal(actual, option) {
			return "", nil
		}
	}
	return "got " + show(actual), nil
}

// checkContains: a string contains a substring, an array contains an element.
func checkContains(actual, expected any, validateOnly bool) (string, error) {
	if validateOnly {
		return "", nil
	}
	switch a := actual.(type) {
	case string:
		sub, ok := expected.(string)
		if !ok {
			return fmt.Sprintf("but a string cannot contain %s", kindOf(expected)), nil
		}
		if !strings.Contains(a, sub) {
			return "got " + show(actual), nil
		}
		return "", nil
	case []any:
		for _, item := range a {
			if equal(item, expected) {
				return "", nil
			}
		}
		return "got " + show(actual), nil
	default:
		return fmt.Sprintf("but it is %s, not a string or an array: %s", kindOf(actual), show(actual)), nil
	}
}

func checkMatches(actual, expected any, validateOnly bool) (string, error) {
	pattern, ok := expected.(string)
	if !ok {
		return "", fmt.Errorf("expected value must be a regular expression string, got %s", kindOf(expected))
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regular expression %q: %w", pattern, err)
	}
	if validateOnly {
		return "", nil
	}
	s, ok := actual.(string)
	if !ok {
		return fmt.Sprintf("but it is %s, not a string: %s", kindOf(actual), show(actual)), nil
	}
	if !re.MatchString(s) {
		return "got " + show(actual), nil
	}
	return "", nil
}

// compare builds the check of an ordering operator; holds tells whether the
// result of comparing actual with expected (-1, 0, +1) satisfies it.
func compare(holds func(cmp int) bool) checkFunc {
	return func(actual, expected any, validateOnly bool) (string, error) {
		want, ok := toNumber(expected)
		if !ok {
			return "", fmt.Errorf("expected value must be a number, got %s", kindOf(expected))
		}
		if validateOnly {
			return "", nil
		}
		got, ok := toNumber(actual)
		if !ok {
			return fmt.Sprintf("but it is %s, not a number: %s", kindOf(actual), show(actual)), nil
		}
		if !holds(got.Cmp(want)) {
			return "got " + show(actual), nil
		}
		return "", nil
	}
}

// checkLength: the number of characters of a string, of elements of an
// array, of fields of an object.
func checkLength(actual, expected any, validateOnly bool) (string, error) {
	want, ok := toNumber(expected)
	if !ok || !want.IsInt() || want.Sign() < 0 {
		return "", fmt.Errorf("expected value must be a non-negative integer, got %s", show(expected))
	}
	if validateOnly {
		return "", nil
	}

	var n int
	switch a := actual.(type) {
	case string:
		n = utf8.RuneCountInString(a)
	case []any:
		n = len(a)
	case map[string]any:
		n = len(a)
	default:
		return fmt.Sprintf("but it is %s, which has no length: %s", kindOf(actual), show(actual)), nil
	}
	if want.Num().IsInt64() && want.Num().Int64() == int64(n) {
		return "", nil
	}
	return fmt.Sprintf("got length %d", n), nil
}

// describe renders the expectation for a message: "to equal 201",
// "to be one of [200,201]".
func describe(op model.AssertOp, expected any) string {
	switch op {
	case model.OpEquals:
		return "to equal " + show(expected)
	case model.OpNotEquals:
		return "not to equal " + show(expected)
	case model.OpIn:
		return "to be one of " + show(expected)
	case model.OpContains:
		return "to contain " + show(expected)
	case model.OpMatches:
		return "to match " + show(expected)
	case model.OpLt:
		return "to be less than " + show(expected)
	case model.OpLte:
		return "to be at most " + show(expected)
	case model.OpGt:
		return "to be greater than " + show(expected)
	case model.OpGte:
		return "to be at least " + show(expected)
	case model.OpLength:
		return "to have length " + show(expected)
	default:
		return string(op) + " " + show(expected)
	}
}

// typeHint points out the usual cause of a confusing "expected 57, got
// "57"": the values look alike but have different types.
func typeHint(actual, expected any) string {
	if ka, ke := kindOf(actual), kindOf(expected); ka != ke {
		return fmt.Sprintf(" (%s, expected %s)", ka, ke)
	}
	return ""
}

// maxShown limits a value in a message: a whole response body in an error
// line helps nobody; the full value is in the report.
const maxShown = 200

// show renders a value as compact JSON for a message.
func show(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	if len(data) > maxShown {
		cut := maxShown
		for cut > 0 && !utf8.RuneStart(data[cut]) {
			cut--
		}
		return string(data[:cut]) + "…"
	}
	return string(data)
}
