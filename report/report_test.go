package report_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tapirtests/core/report"
)

// Short names for building trees in tests.
const (
	P = report.Passed
	F = report.Failed
	E = report.Error
	S = report.Skipped
)

func step(status report.Status) *report.Step { return &report.Step{Status: status} }

func ignored(status report.Status) *report.Step {
	return &report.Step{Status: status, Ignored: true}
}

// scenario returns a scenario with the given steps, rolled up.
func scenario(steps ...*report.Step) *report.Scenario {
	sc := &report.Scenario{Steps: steps}
	sc.Rollup()
	return sc
}

// done returns a scenario that already has a status, as if rolled up.
func done(status report.Status) *report.Scenario { return &report.Scenario{Status: status} }

func TestStepFailed(t *testing.T) {
	tests := []struct {
		step *report.Step
		want bool
	}{
		{step(P), false},
		{step(S), false},
		{step(F), true},
		{step(E), true},
		{ignored(F), false},
		{ignored(E), false},
		{ignored(P), false},
	}
	for _, tt := range tests {
		if got := tt.step.Failed(); got != tt.want {
			t.Errorf("Step{%s, ignored: %v}.Failed() = %v, want %v", tt.step.Status, tt.step.Ignored, got, tt.want)
		}
	}
}

func TestScenarioRollup(t *testing.T) {
	tests := []struct {
		name string
		sc   *report.Scenario
		want report.Status
	}{
		{"no steps", &report.Scenario{}, P},
		{"all passed", &report.Scenario{Steps: []*report.Step{step(P), step(P)}}, P},
		{"a step failed", &report.Scenario{Steps: []*report.Step{step(P), step(F), step(S)}}, F},
		{"a step errored", &report.Scenario{Steps: []*report.Step{step(P), step(E), step(S)}}, E},
		{"ignored failure does not fail the scenario", &report.Scenario{Steps: []*report.Step{step(P), ignored(F), step(P)}}, P},
		{"ignored error does not fail the scenario", &report.Scenario{Steps: []*report.Step{ignored(E), step(P)}}, P},
		{"only ignored failures", &report.Scenario{Steps: []*report.Step{ignored(F), ignored(E)}}, P},
		{
			// The step that stopped the scenario decides, not the worst one.
			"the first unignored failure decides",
			&report.Scenario{Steps: []*report.Step{ignored(E), step(F), step(S)}}, F,
		},
		{"error of the scenario itself", &report.Scenario{Error: "input \"token\" is not provided"}, E},
		{
			"error of the scenario itself wins over its steps",
			&report.Scenario{Error: "output could not be computed", Steps: []*report.Step{step(P), ignored(F)}}, E,
		},
		{"skipped stays skipped", &report.Scenario{Status: S}, S},
		{"skipped stays skipped with skipped steps", &report.Scenario{Status: S, Steps: []*report.Step{step(S)}}, S},
		{
			// Rollup replaces a status set earlier, except Skipped.
			"recomputed from steps",
			&report.Scenario{Status: F, Steps: []*report.Step{step(P)}}, P,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sc.Rollup(); got != tt.want {
				t.Errorf("Rollup() = %s, want %s", got, tt.want)
			}
			if tt.sc.Status != tt.want {
				t.Errorf("Status = %s, want %s", tt.sc.Status, tt.want)
			}
		})
	}
}

func TestGroupRollup(t *testing.T) {
	tests := []struct {
		name         string
		group        *report.Group
		want         report.Status
		tearDownFail bool
	}{
		{"empty group", &report.Group{}, P, false},
		{"all passed", &report.Group{Setup: []*report.Scenario{done(P)}, Main: []*report.Scenario{done(P), done(P)}}, P, false},
		{
			"setup failed, main skipped",
			&report.Group{Setup: []*report.Scenario{done(F)}, Main: []*report.Scenario{done(S)}}, F, false,
		},
		{
			"main scenario failed, the rest skipped",
			&report.Group{Main: []*report.Scenario{done(P), done(F), done(S)}}, F, false,
		},
		{"main scenario errored", &report.Group{Main: []*report.Scenario{done(E)}}, E, false},
		{
			"error is worse than failure, whatever the order",
			&report.Group{Setup: []*report.Scenario{done(P)}, Main: []*report.Scenario{done(F)},
				Groups: []*report.Group{{Status: E}}}, E, false,
		},
		{
			// Sibling groups are independent: one failing does not stop the
			// others, but the parent is failed.
			"a nested group failed, its siblings passed",
			&report.Group{Main: []*report.Scenario{done(P)},
				Groups: []*report.Group{{Status: P}, {Status: F}, {Status: P}}}, F, false,
		},
		{"nested groups passed", &report.Group{Groups: []*report.Group{{Status: P}, {Status: P}}}, P, false},
		{
			"skipped children do not count",
			&report.Group{Main: []*report.Scenario{done(P), done(S)}, Groups: []*report.Group{{Status: S}}}, P, false,
		},

		// Tear-down is a warning, not a failure.
		{
			"tear-down failed, everything else passed",
			&report.Group{Main: []*report.Scenario{done(P)}, TearDown: []*report.Scenario{done(F)}}, P, true,
		},
		{
			"tear-down errored",
			&report.Group{Main: []*report.Scenario{done(P)}, TearDown: []*report.Scenario{done(P), done(E)}}, P, true,
		},
		{
			"tear-down failed after a failed main",
			&report.Group{Main: []*report.Scenario{done(F)}, TearDown: []*report.Scenario{done(F)}}, F, true,
		},
		{
			"tear-down passed after a failed main",
			&report.Group{Main: []*report.Scenario{done(F)}, TearDown: []*report.Scenario{done(P)}}, F, false,
		},
		{
			"skipped tear-down is not a failure",
			&report.Group{TearDown: []*report.Scenario{done(S)}}, P, false,
		},

		// The group itself.
		{"error of the group itself", &report.Group{Error: "variable could not be computed"}, E, false},
		{
			"error of the group itself wins",
			&report.Group{Error: "x", Main: []*report.Scenario{done(S)}, TearDown: []*report.Scenario{done(P)}}, E, false,
		},
		{"skipped stays skipped", &report.Group{Status: S, Main: []*report.Scenario{done(S)}}, S, false},
		{
			// A stale flag from an earlier rollup is cleared.
			"tear-down flag is recomputed",
			&report.Group{TearDownFailed: true, TearDown: []*report.Scenario{done(P)}}, P, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.Rollup(); got != tt.want {
				t.Errorf("Rollup() = %s, want %s", got, tt.want)
			}
			if tt.group.TearDownFailed != tt.tearDownFail {
				t.Errorf("TearDownFailed = %v, want %v", tt.group.TearDownFailed, tt.tearDownFail)
			}
		})
	}
}

func TestRunRollup(t *testing.T) {
	// /root
	//   setup:    passed (1 step)
	//   main:     passed with an ignored failure (3 steps)
	//   /root/products
	//     main:     failed (passed, failed, skipped), then a skipped scenario
	//     tearDown: failed (1 step errored)
	//   /root/orders
	//     main:     passed (2 steps)
	//   tearDown: passed (1 step)
	products := &report.Group{
		Name: "products", Path: "/root/products",
		Main: []*report.Scenario{
			scenario(step(P), step(F), step(S)),
			{Status: S, Steps: []*report.Step{step(S)}},
		},
		TearDown: []*report.Scenario{scenario(step(E))},
	}
	orders := &report.Group{
		Name: "orders", Path: "/root/orders",
		Main: []*report.Scenario{scenario(step(P), step(P))},
	}
	root := &report.Group{
		Name: "root", Path: "/root",
		Setup:    []*report.Scenario{scenario(step(P))},
		Main:     []*report.Scenario{scenario(step(P), ignored(F), step(P))},
		Groups:   []*report.Group{products, orders},
		TearDown: []*report.Scenario{scenario(step(P))},
	}
	products.Rollup()
	orders.Rollup()
	root.Rollup()

	run := &report.Run{Root: root}
	if got := run.Rollup(); got != F {
		t.Errorf("run status = %s, want %s", got, F)
	}
	if run.OK() {
		t.Error("a failed run reports OK")
	}
	if products.Status != F || !products.TearDownFailed {
		t.Errorf("products: status %s, tearDownFailed %v", products.Status, products.TearDownFailed)
	}
	if orders.Status != P {
		t.Errorf("orders: status %s: a failed sibling must not affect it", orders.Status)
	}

	want := report.Summary{
		// setup, main, products main x2, products tearDown, orders main, root tearDown
		Scenarios: report.Counts{Passed: 4, Failed: 1, Error: 1, Skipped: 1},
		// The ignored failure is counted separately, not as a failed step.
		Steps:            report.Counts{Passed: 7, Failed: 1, Error: 1, Skipped: 2},
		IgnoredSteps:     1,
		TearDownFailures: 1,
	}
	if run.Summary != want {
		t.Errorf("summary\n got  %+v\n want %+v", run.Summary, want)
	}

	// Rollup is repeatable: it recomputes, it does not accumulate.
	run.Rollup()
	if run.Summary != want {
		t.Errorf("summary after a second rollup\n got  %+v\n want %+v", run.Summary, want)
	}
}

func TestRunRollupOfIsolatedScenario(t *testing.T) {
	run := &report.Run{Scenario: scenario(step(P), step(E))}
	if got := run.Rollup(); got != E {
		t.Errorf("status = %s, want %s", got, E)
	}
	want := report.Summary{Scenarios: report.Counts{Error: 1}, Steps: report.Counts{Passed: 1, Error: 1}}
	if run.Summary != want {
		t.Errorf("summary = %+v, want %+v", run.Summary, want)
	}
}

func TestRunOK(t *testing.T) {
	for status, want := range map[report.Status]bool{P: true, S: true, F: false, E: false} {
		run := &report.Run{Root: &report.Group{Status: status}}
		run.Rollup()
		if got := run.OK(); got != want {
			t.Errorf("run with a %s root: OK() = %v, want %v", status, got, want)
		}
	}
	if empty := (&report.Run{}); empty.Rollup() != P || !empty.OK() {
		t.Error("an empty run is not OK")
	}
}

// A tear-down failure alone does not fail the run, but it is visible.
func TestTearDownFailureDoesNotFailTheRun(t *testing.T) {
	root := &report.Group{
		Main:     []*report.Scenario{scenario(step(P))},
		TearDown: []*report.Scenario{scenario(step(F))},
	}
	root.Rollup()
	run := &report.Run{Root: root}
	run.Rollup()

	if !run.OK() {
		t.Error("a tear-down failure fails the run")
	}
	if !root.TearDownFailed || run.Summary.TearDownFailures != 1 {
		t.Errorf("tear-down failure is not visible: flag %v, count %d", root.TearDownFailed, run.Summary.TearDownFailures)
	}
}

func TestStatusIsFailure(t *testing.T) {
	for status, want := range map[report.Status]bool{P: false, S: false, F: true, E: true, "": false} {
		if got := status.IsFailure(); got != want {
			t.Errorf("%q.IsFailure() = %v, want %v", status, got, want)
		}
	}
}

func TestJSON(t *testing.T) {
	started := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	run := &report.Run{
		Target:     "/root",
		Seed:       81723,
		StartedAt:  started,
		FinishedAt: started.Add(1500 * time.Millisecond),
		Duration:   report.Duration(1500 * time.Millisecond),
		Root: &report.Group{
			Name: "root", Path: "/root", Duration: report.Duration(1500 * time.Millisecond),
			Main: []*report.Scenario{{
				Alias: "create", ScenarioID: "createProduct", Duration: report.Duration(84 * time.Millisecond),
				Steps: []*report.Step{{
					ID: "create", RequestID: "createProduct", Status: report.Failed,
					Duration: report.Duration(84*time.Millisecond + 700*time.Microsecond),
					Request:  &report.Request{Line: "POST http://api/products", Body: map[string]any{"name": "Phone"}},
					Response: &report.Response{Status: 500, Duration: report.Duration(84 * time.Millisecond)},
					Assertions: []report.Assertion{
						{Path: "$.status", Op: "equals", Expected: json.Number("201"), Actual: json.Number("500"), Found: true,
							Message: "expected $.status to equal 201, got 500"},
						{Path: "$.body.id", Op: "exists", Message: "expected $.body.id to exist, but there is nothing at this path"},
					},
				}},
			}},
		},
	}
	run.Root.Main[0].Rollup()
	run.Root.Rollup()
	run.Rollup()

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	const want = `{
  "status": "failed",
  "target": "/root",
  "seed": 81723,
  "startedAt": "2026-10-07T12:00:00Z",
  "finishedAt": "2026-10-07T12:00:01.5Z",
  "durationMs": 1500,
  "summary": {
    "scenarios": {
      "passed": 0,
      "failed": 1,
      "error": 0,
      "skipped": 0
    },
    "steps": {
      "passed": 0,
      "failed": 1,
      "error": 0,
      "skipped": 0
    },
    "ignoredSteps": 0,
    "tearDownFailures": 0
  },
  "root": {
    "name": "root",
    "path": "/root",
    "status": "failed",
    "durationMs": 1500,
    "main": [
      {
        "alias": "create",
        "scenarioId": "createProduct",
        "status": "failed",
        "durationMs": 84,
        "steps": [
          {
            "id": "create",
            "requestId": "createProduct",
            "status": "failed",
            "durationMs": 84,
            "request": {
              "line": "POST http://api/products",
              "body": {
                "name": "Phone"
              }
            },
            "response": {
              "status": 500,
              "durationMs": 84
            },
            "assertions": [
              {
                "path": "$.status",
                "op": "equals",
                "passed": false,
                "expected": 201,
                "actual": 500,
                "found": true,
                "message": "expected $.status to equal 201, got 500"
              },
              {
                "path": "$.body.id",
                "op": "exists",
                "passed": false,
                "found": false,
                "message": "expected $.body.id to exist, but there is nothing at this path"
              }
            ]
          }
        ]
      }
    ]
  }
}`
	if string(data) != want {
		t.Errorf("JSON\n got:\n%s\n want:\n%s", data, want)
	}

	// The report reads back: the site loads report.json produced by the CLI.
	var back report.Run
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Status != report.Failed || back.Duration != report.Duration(1500*time.Millisecond) {
		t.Errorf("read back: status %s, duration %v", back.Status, time.Duration(back.Duration))
	}
	if got := back.Root.Main[0].Steps[0].Assertions[0].Message; got != "expected $.status to equal 201, got 500" {
		t.Errorf("read back assertion message = %q", got)
	}
}
