// Package report describes the result of a run: a tree of groups, scenarios
// and steps with everything needed to see what was sent, what came back and
// why something failed.
//
// The engine fills the tree while it runs; shells print it, write it to
// report.json or show it in the UI. The package holds data and the rules
// that fold the statuses of steps into the statuses of scenarios, groups and
// the run. Values stored here are already masked: the engine removes secrets
// before they get into a report.
package report

import (
	"encoding/json"
	"time"
)

// Status is the outcome of a step, a scenario, a group or a run.
type Status string

const (
	// Passed: everything that ran did what was expected.
	Passed Status = "passed"
	// Failed: the API did not behave as expected — an assertion did not hold.
	Failed Status = "failed"
	// Error: the check itself could not be carried out — a template did not
	// resolve, the server was unreachable, a call timed out, an input was
	// not provided. It points to a problem of the test or its environment
	// rather than of the API.
	Error Status = "error"
	// Skipped: it did not run because something before it failed.
	Skipped Status = "skipped"
)

// rank orders statuses from harmless to severe, to pick the worst of many.
func (s Status) rank() int {
	switch s {
	case Error:
		return 3
	case Failed:
		return 2
	case Passed:
		return 1
	default:
		return 0
	}
}

// worse returns the more severe of two statuses.
func worse(a, b Status) Status {
	if b.rank() > a.rank() {
		return b
	}
	return a
}

// IsFailure reports whether the status means something went wrong.
func (s Status) IsFailure() bool { return s == Failed || s == Error }

// Duration is a time span written to JSON as a whole number of milliseconds.
type Duration time.Duration

// MarshalJSON encodes the duration as milliseconds.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).Milliseconds())
}

// UnmarshalJSON decodes a duration from milliseconds.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var ms int64
	if err := json.Unmarshal(data, &ms); err != nil {
		return err
	}
	*d = Duration(time.Duration(ms) * time.Millisecond)
	return nil
}

// Run is the result of one run.
type Run struct {
	Status Status `json:"status"`
	// Target is what was run: a group path ("/root", "/root/products") or a
	// scenario ID for an isolated run of a scenario.
	Target string `json:"target"`
	// Isolated tells that the target ran without its parent groups.
	Isolated bool `json:"isolated,omitempty"`
	// Seed reproduces the random data of the run.
	Seed       uint64    `json:"seed"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Duration   Duration  `json:"durationMs"`
	Summary    Summary   `json:"summary"`

	// Root is the tree of group results, from the root group down to the
	// target; nil for an isolated run of a scenario.
	Root *Group `json:"root,omitempty"`
	// Scenario is the result of an isolated run of a scenario.
	Scenario *Scenario `json:"scenario,omitempty"`
}

// Summary counts what happened in a run.
type Summary struct {
	Scenarios Counts `json:"scenarios"`
	Steps     Counts `json:"steps"`
	// IgnoredSteps is how many steps failed but were told to be ignored;
	// they are not counted in Steps.Failed or Steps.Error.
	IgnoredSteps int `json:"ignoredSteps"`
	// TearDownFailures is how many tear-down scenarios failed. They do not
	// make the run fail, but leftovers on the server may break later runs.
	TearDownFailures int `json:"tearDownFailures"`
}

// Counts is a number of things per status.
type Counts struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Error   int `json:"error"`
	Skipped int `json:"skipped"`
}

func (c *Counts) add(s Status) {
	switch s {
	case Passed:
		c.Passed++
	case Failed:
		c.Failed++
	case Error:
		c.Error++
	case Skipped:
		c.Skipped++
	}
}

// Group is the result of one group.
type Group struct {
	Name   string `json:"name"`
	Path   string `json:"path"` // "/root/products"
	Status Status `json:"status"`
	// Error is a problem of the group itself, outside its scenarios: its
	// initial variables could not be computed.
	Error string `json:"error,omitempty"`
	// TearDownFailed tells that a tear-down scenario failed. It is a
	// warning: it does not change Status.
	TearDownFailed bool     `json:"tearDownFailed,omitempty"`
	Duration       Duration `json:"durationMs"`

	Setup    []*Scenario `json:"setup,omitempty"`
	Main     []*Scenario `json:"main,omitempty"`
	Groups   []*Group    `json:"groups,omitempty"`
	TearDown []*Scenario `json:"tearDown,omitempty"`
}

// Scenario is the result of one scenario call.
type Scenario struct {
	Alias      string `json:"alias"`
	ScenarioID string `json:"scenarioId"`
	Status     Status `json:"status"`
	// Error is a problem of the call outside its steps: an input was not
	// provided, an output could not be computed, the scenario is unknown.
	Error    string   `json:"error,omitempty"`
	Duration Duration `json:"durationMs"`
	Steps    []*Step  `json:"steps,omitempty"`
}

// Step is the result of one step.
type Step struct {
	ID        string `json:"id"`
	RequestID string `json:"requestId"`
	Status    Status `json:"status"`
	// Ignored tells that the step failed but was told not to stop the
	// scenario (ignoreError); Status still shows what happened.
	Ignored bool `json:"ignored,omitempty"`
	// Error explains the Error status: why no assertions could be checked.
	Error    string   `json:"error,omitempty"`
	Duration Duration `json:"durationMs"`

	// Request is what was sent; nil if the request could not be built.
	Request *Request `json:"request,omitempty"`
	// Response is what came back; nil if there was no response.
	Response *Response `json:"response,omitempty"`
	// Assertions are the results of all assertions of the step: every one is
	// evaluated, not only the first that fails.
	Assertions []Assertion `json:"assertions,omitempty"`
	// Extracted are the variables the step saved to the scenario scope.
	Extracted map[string]any `json:"extracted,omitempty"`
}

// Request is a request as it was sent.
type Request struct {
	Line    string            `json:"line"` // "POST http://api:8080/products"
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

// Response is a received response.
type Response struct {
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     any               `json:"body,omitempty"`
	Duration Duration          `json:"durationMs"`
}

// Assertion is the result of one assertion.
type Assertion struct {
	Path     string `json:"path"`
	Op       string `json:"op"`
	Passed   bool   `json:"passed"`
	Expected any    `json:"expected,omitempty"`
	// Actual is the value found at Path; Found tells whether there was one,
	// since a found null and nothing at all both have a nil Actual.
	Actual any  `json:"actual,omitempty"`
	Found  bool `json:"found"`
	// Message explains a failed assertion, or says why it could not be
	// evaluated.
	Message string `json:"message,omitempty"`
}
