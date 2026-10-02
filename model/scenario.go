package model

// ScenarioDef is the definition of a scenario, stored in the Scenario bucket.
//
// A scenario works like a function: it declares inputs and outputs, runs its
// steps in order in its own variable scope and returns the outputs. A failed
// step stops the scenario unless the step ignores errors. The variable scope
// exists only while the scenario runs and is created by the engine.
type ScenarioDef struct {
	ID          ScenarioID // identifier
	Name        string     // scenario name
	Description string     // scenario description
	Inputs      []Param    // input fields (from user form)
	Outputs     []Output   // output fields
	Steps       []Step     // steps in order
	GeneratedBy string     // who created the scenario: "" (user), "heuristic", later "llm"
}

// ScenarioCall is a usage of a ScenarioDef inside a group section.
//
// It passes inputs to the scenario and saves its outputs to the group scope.
// Inputs not listed here are taken from the group scope by the same name.
// Outputs are saved as "<alias>.<output>" unless renamed in Outputs; a rename
// may intentionally overwrite an existing variable (last write wins).
type ScenarioCall struct {
	Alias      string            // unique in group scope; on empty use scenarioID
	ScenarioID ScenarioID        // scenario identifier
	Inputs     map[string]Value  // fields, that pass into the scenario
	Outputs    map[string]string // fields, that extract from the scenario. Rename
}

// GetAlias returns the alias of the call, or the scenario ID if the alias is
// empty.
func (sc ScenarioCall) GetAlias() string {
	if sc.Alias == "" {
		return sc.ScenarioID.String()
	}
	return sc.Alias
}

// Param is one input of a ScenarioDef signature.
type Param struct {
	Name        string // variable name inside the scenario scope
	Required    bool   // a call must provide this input, directly or from the group scope
	Default     Value  // value used when the input is not provided; nil means no default
	Description string // description
}

// Output is one output of a ScenarioDef signature.
type Output struct {
	Name  string // output name, visible to the caller
	Value Value  // template evaluated in the scenario scope when it finishes, usually "{{productId}}"
}

// ScenarioID identifies a ScenarioDef in the Scenario bucket.
type ScenarioID string

// String implements fmt.Stringer.
func (sc ScenarioID) String() string {
	return string(sc)
}
