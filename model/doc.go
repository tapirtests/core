// Package model describes a Tapir project: the entities users write in
// tapir.json and the API operations imported from a specification.
//
// The model separates definitions from usages:
//
//   - A definition is a reusable, unchanging form stored in a bucket:
//     RequestDef (Requests bucket) and ScenarioDef (Scenario bucket). It fixes
//     the contract — which inputs are accepted and which outputs are produced.
//   - A usage is a concrete call of a definition in a concrete place, with
//     concrete values and expectations: RequestCall (a step inside a scenario)
//     and ScenarioCall (a scenario inside a group section).
//
// Groups form a tree rooted at Project.Root ("/root"). Each group has Setup,
// Main and TearDown sections; running a group cascades Setup from the root
// down and TearDown back up.
//
// The model is plain data. It contains no execution logic and no I/O, and it
// is never modified during a run: runtime state (variable scopes, results)
// lives in the engine, so one Project can be run many times, even concurrently.
package model
