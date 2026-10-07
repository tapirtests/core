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
	// Setup, Main and TearDown write to the same group scope, so aliases
	// must be unique across all three sections.
	names := &groupNames{aliases: make(map[string]bool)}
	v.validateCalls(g.Setup, ptr.Key("setup"), names)
	v.validateCalls(g.Main.Scenarios, ptr.Key("main").Key("scenarios"), names)
	v.validateCalls(g.TearDown, ptr.Key("tearDown"), names)

	for name, value := range g.Vars {
		varPtr := ptr.Key("vars").Key(name)
		if !identifierPattern.MatchString(name) {
			v.errorf(V0807, varPtr, "group variable name %q must match %s", name, identifierPattern)
		}
		v.validateNotReserved(name, varPtr, "group variable")
		v.validateTemplates(value, varPtr)
		names.variables = append(names.variables, declaredVar{name, varPtr})
	}

	// Outputs of a call are stored in the group scope as an object named by
	// the alias, so a plain variable with the same name and the alias would
	// overwrite each other.
	for _, variable := range names.variables {
		if names.aliases[variable.name] {
			v.errorf(V0815, variable.ptr,
				"variable %q clashes with the alias of a scenario call in this group", variable.name)
		}
	}

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

// groupNames collects the names that end up in the scope of one group, to
// find clashes between them once all sections were checked.
type groupNames struct {
	aliases   map[string]bool // effective aliases of scenario calls
	variables []declaredVar   // group variables and renamed outputs
}

// declaredVar is a plain variable of a group scope and where it is declared.
type declaredVar struct {
	name string
	ptr  diag.Pointer
}

// validateCalls checks the scenario calls of one group section. names is
// shared by all sections of the group.
func (v *validator) validateCalls(calls []model.ScenarioCall, ptr diag.Pointer, names *groupNames) {
	for i, c := range calls {
		v.validateCall(c, ptr.Index(i), names)
	}
}

// validateCall checks a scenario call: the reference, the alias, and that
// passed inputs and renamed outputs exist in the scenario signature.
//
// Required inputs that are not passed are not reported here: they may come
// from the group scope, which is known only after analyzing the whole tree.
func (v *validator) validateCall(c model.ScenarioCall, ptr diag.Pointer, names *groupNames) {
	v.validateAlias(c, ptr.Key("alias"), names.aliases)

	// Names and templates are checked whether the scenario exists or not.
	for name, value := range c.Inputs {
		v.validateTemplates(value, ptr.Key("inputs").Key(name))
	}
	for output, newName := range c.Outputs {
		outPtr := ptr.Key("outputs").Key(output)
		if !identifierPattern.MatchString(newName) {
			v.errorf(V0814, outPtr, "new name %q of output %q must match %s", newName, output, identifierPattern)
		}
		v.validateNotReserved(newName, outPtr, "variable")
		names.variables = append(names.variables, declaredVar{newName, outPtr})
	}

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
	for output := range c.Outputs {
		if !outputs[output] {
			v.errorf(V0813, ptr.Key("outputs").Key(output), "scenario %q has no output %q", sc.ID, output)
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
	v.validateNotReserved(alias, ptr, "alias")
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
