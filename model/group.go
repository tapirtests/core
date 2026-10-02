package model

// Group - description of the group. The group contains three sections: setup, main and teardown.
// The Setup and teardown parts store only sets of scripts, while the main part stores a set of scripts
// and nested groups.
// The Setup part performs first of all. When the script crashes in the setup part,
// the group stops executing, and execution immediately proceeds to the teardown part.
// The main part is executed only after successful completion of the setup part. When a script crashes
// in the main part, execution stops and goes straight to the teardown part. A failed nested group
// does not stop its sibling groups: they still run, and the failure is reported.
// The teardown part is executed anyway, if the setup part has started. When the script crashes in the teardown part,
// execution does not stop. Teardown execution cannot be interrupted.
// The group also has a local variable storage (Vars). The scenarios of the group read and write it.
// Nested groups only read it: whatever they write stays in their own storage and is not visible
// to the parent or to sibling groups.
type Group struct {
	Name        string           // unique among sibling groups; the root group is named "root"
	Description string           // human-readable purpose of the group
	Vars        map[string]Value // initial values of the group scope, as written in tapir.json
	Setup       []ScenarioCall   // setup scenarios
	Main        Main             // main part (scenarios and nested groups)
	TearDown    []ScenarioCall   // teardown scenarios
}

// Main is the main section of a group. Its scenarios run first, in order,
// then its nested groups, in order.
type Main struct {
	Scenarios []ScenarioCall // main scenarios
	Groups    []*Group       // nested groups
}
