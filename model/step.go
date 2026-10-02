package model

// Step is one step of a scenario. The set of step kinds is closed: only types
// of this package can implement Step. RequestCall is the only kind for now;
// streaming protocols will add another one.
type Step interface {
	// StepID returns the identifier of the step, unique within its scenario.
	StepID() string
	isStep()
}

// Compile-time checks that every step kind implements Step.
var _ Step = (*RequestCall)(nil)
