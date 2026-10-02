package validate

import (
	"regexp"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
)

// groupNamePattern is the allowed form of a group name. Names make up group
// paths like /root/orders/payments used by the UI and the CLI, so they must
// not contain "/" or spaces.
var groupNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validateGroups walks the group tree from the root. The presence and the
// name of the root group are checked with the project.
func (v *validator) validateGroups() {
	if v.p.Root == nil {
		return
	}
	visited := make(map[*model.Group]bool)
	visited[v.p.Root] = true
	v.validateGroup(v.p.Root, diag.Root.Key("root"), visited)
}

// validateGroup checks one group and descends into its nested groups.
// visited holds the groups already met in the tree: the model is built of
// pointers, so a group may be placed into its own subtree or into two places
// by mistake. Such a group is reported and not entered again, which also
// keeps the walk from looping forever.
func (v *validator) validateGroup(g *model.Group, ptr diag.Pointer, visited map[*model.Group]bool) {
	for name := range g.Vars {
		if !identifierPattern.MatchString(name) {
			v.errorf(V0807, ptr.Key("vars").Key(name), "group variable name %q must match %s",
				name, identifierPattern)
		}
	}

	// Setup, Main and TearDown write to the same group scope, so aliases
	// must be unique across all three sections.
	aliases := make(map[string]bool)
	v.validateCalls(g.Setup, ptr.Key("setup"), aliases)
	v.validateCalls(g.Main.Scenarios, ptr.Key("main").Key("scenarios"), aliases)
	v.validateCalls(g.TearDown, ptr.Key("tearDown"), aliases)

	siblings := make(map[string]bool, len(g.Main.Groups))
	for i, child := range g.Main.Groups {
		childPtr := ptr.Key("main").Key("groups").Index(i)
		switch {
		case child == nil:
			v.errorf(V0802, childPtr, "group is empty")
			continue
		case visited[child]:
			v.errorf(V0806, childPtr,
				"group %q already appears in the tree: a group cannot be nested into itself or placed twice",
				child.Name)
			continue
		}
		visited[child] = true

		v.validateGroupName(child.Name, siblings, childPtr.Key("name"))
		v.validateGroup(child, childPtr, visited)
	}
}

// validateGroupName checks the name of a nested group: it is set, fits into
// a group path and is unique among its siblings.
func (v *validator) validateGroupName(name string, siblings map[string]bool, ptr diag.Pointer) {
	switch {
	case name == "":
		v.errorf(V0803, ptr, "group name is required")
		return
	case !groupNamePattern.MatchString(name):
		v.errorf(V0804, ptr, "group name %q must match %s", name, groupNamePattern)
	}
	if siblings[name] {
		v.errorf(V0805, ptr, "group name %q is already used by a sibling group", name)
	}
	siblings[name] = true
}

// validateCalls checks the scenario calls of one group section. aliases is
// shared by all sections of the group.
func (v *validator) validateCalls(calls []model.ScenarioCall, ptr diag.Pointer, aliases map[string]bool) {
	for i, c := range calls {
		v.validateCall(c, ptr.Index(i), aliases)
	}
}

// validateCall checks a scenario call: the reference, the alias, and that
// passed inputs and renamed outputs exist in the scenario signature.
//
// Required inputs that are not passed are not reported here: they may come
// from the group scope, which is known only after analyzing the whole tree.
func (v *validator) validateCall(c model.ScenarioCall, ptr diag.Pointer, aliases map[string]bool) {
	v.validateAlias(c, ptr.Key("alias"), aliases)

	sc, ok := v.lookupScenario(c.ScenarioID, ptr.Key("scenarioId"))
	if !ok {
		return
	}

	inputs := make(map[string]bool, len(sc.Inputs))
	for _, in := range sc.Inputs {
		inputs[in.Name] = true
	}
	for name := range c.Inputs {
		if !inputs[name] {
			v.errorf(V0812, ptr.Key("inputs").Key(name), "scenario %q has no input %q", sc.ID, name)
		}
	}

	outputs := make(map[string]bool, len(sc.Outputs))
	for _, out := range sc.Outputs {
		outputs[out.Name] = true
	}
	for output, newName := range c.Outputs {
		outPtr := ptr.Key("outputs").Key(output)
		if !outputs[output] {
			v.errorf(V0813, outPtr, "scenario %q has no output %q", sc.ID, output)
		}
		if !identifierPattern.MatchString(newName) {
			v.errorf(V0814, outPtr, "new name %q of output %q must match %s", newName, output, identifierPattern)
		}
	}
}

// validateAlias checks the alias of a call. An empty alias falls back to the
// scenario ID; the effective alias must be unique in the group, since outputs
// are saved as "<alias>.<output>".
func (v *validator) validateAlias(c model.ScenarioCall, ptr diag.Pointer, aliases map[string]bool) {
	if c.Alias != "" && !identifierPattern.MatchString(c.Alias) {
		v.errorf(V0810, ptr, "alias %q must match %s", c.Alias, identifierPattern)
	}

	alias := c.GetAlias()
	if alias == "" {
		return // no scenario ID either; reported with the reference
	}
	if aliases[alias] {
		v.errorf(V0811, ptr, "alias %q is already used in the group; set a distinct alias", alias)
	}
	aliases[alias] = true
}

// lookupScenario returns the scenario a call refers to. ok is false when the
// signature cannot be checked: the reference is missing or broken (reported
// here), or the scenario entry itself is nil (reported with scenarios).
func (v *validator) lookupScenario(id model.ScenarioID, ptr diag.Pointer) (*model.ScenarioDef, bool) {
	if id == "" {
		v.errorf(V0808, ptr, "scenario ID is required")
		return nil, false
	}
	sc, found := v.p.Scenarios[id]
	if !found {
		v.errorf(V0809, ptr, "unknown scenario %q", id)
		return nil, false
	}
	return sc, sc != nil
}
