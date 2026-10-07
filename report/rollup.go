package report

// The rules that fold statuses upwards. The engine sets the status of every
// step and marks what did not run as Skipped; everything else is computed
// here, so the rules live in one place and can be tested without running
// anything.

// Failed reports whether the step counts as a failure of its scenario: it
// failed or errored and was not told to be ignored.
func (s *Step) Failed() bool {
	return s.Status.IsFailure() && !s.Ignored
}

// Rollup computes the status of the scenario from its steps and returns it.
//
// A scenario marked Skipped stays skipped. A scenario with an Error of its
// own is Error. Otherwise the scenario has the status of its first step that
// failed without being ignored — the step that stopped it — and is Passed if
// there is none: ignored failures do not make a scenario fail.
func (sc *Scenario) Rollup() Status {
	switch {
	case sc.Status == Skipped:
	case sc.Error != "":
		sc.Status = Error
	default:
		sc.Status = Passed
		for _, step := range sc.Steps {
			if step.Failed() {
				sc.Status = step.Status
				break
			}
		}
	}
	return sc.Status
}

// Rollup computes the status of the group from what it contains and returns
// it. The scenarios and nested groups must be rolled up already.
//
// A group marked Skipped stays skipped. A group with an Error of its own is
// Error. Otherwise the group has the worst status among its setup scenarios,
// main scenarios and nested groups, Error being worse than Failed; skipped
// ones do not count, and an empty group is Passed.
//
// Tear-down scenarios do not affect the status: a failed tear-down sets
// TearDownFailed, a warning that something may be left behind on the server.
func (g *Group) Rollup() Status {
	g.TearDownFailed = false
	for _, sc := range g.TearDown {
		if sc.Status.IsFailure() {
			g.TearDownFailed = true
		}
	}

	switch {
	case g.Status == Skipped:
	case g.Error != "":
		g.Status = Error
	default:
		status := Passed
		for _, sc := range g.Setup {
			status = worse(status, sc.Status)
		}
		for _, sc := range g.Main {
			status = worse(status, sc.Status)
		}
		for _, child := range g.Groups {
			status = worse(status, child.Status)
		}
		g.Status = status
	}
	return g.Status
}

// Rollup computes the status and the summary of the run from its tree,
// which must be rolled up already, and returns the status.
func (r *Run) Rollup() Status {
	r.Summary = Summary{}
	switch {
	case r.Root != nil:
		r.Status = r.Root.Status
		r.Summary.addGroup(r.Root)
	case r.Scenario != nil:
		r.Status = r.Scenario.Status
		r.Summary.addScenario(r.Scenario, false)
	default:
		r.Status = Passed
	}
	return r.Status
}

// OK reports whether the run succeeded: nothing failed and nothing errored.
// A shell turns it into the exit code.
func (r *Run) OK() bool { return !r.Status.IsFailure() }

func (s *Summary) addGroup(g *Group) {
	for _, sc := range g.Setup {
		s.addScenario(sc, false)
	}
	for _, sc := range g.Main {
		s.addScenario(sc, false)
	}
	for _, child := range g.Groups {
		s.addGroup(child)
	}
	for _, sc := range g.TearDown {
		s.addScenario(sc, true)
	}
}

// addScenario counts a scenario and its steps. A failed tear-down scenario
// is counted in TearDownFailures as well: it is a failure of the scenario,
// but not of the run.
func (s *Summary) addScenario(sc *Scenario, tearDown bool) {
	s.Scenarios.add(sc.Status)
	if tearDown && sc.Status.IsFailure() {
		s.TearDownFailures++
	}
	for _, step := range sc.Steps {
		if step.Status.IsFailure() && step.Ignored {
			s.IgnoredSteps++
			continue
		}
		s.Steps.add(step.Status)
	}
}
