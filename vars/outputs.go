package vars

// Output is a computed output of a scenario: its name in the scenario
// signature and its value, already resolved in the scenario scope.
type Output struct {
	Name  string
	Value any
}

// StoreOutputs saves the outputs of a scenario call into the scope of the
// caller (the group scope).
//
// By default, outputs are kept together under the alias of the call, as an
// object: output "token" of the call "login" is read as {{login.token}}.
// Aliases are unique within a group, so by default outputs of different calls
// never overwrite each other.
//
// rename maps an output name to a plain variable name: with
// {"token": "sellerToken"} the output is stored as {{sellerToken}} instead.
// A renamed output may deliberately overwrite an existing variable; outputs
// are stored in the given order, so if two of them get the same name, the
// last one wins.
//
// If every output is renamed (or there are none), no alias object is stored.
func StoreOutputs(scope *Scope, alias string, outputs []Output, rename map[string]string) {
	var grouped map[string]any
	for _, out := range outputs {
		if name, ok := rename[out.Name]; ok {
			scope.Set(name, out.Value)
			continue
		}
		if grouped == nil {
			grouped = make(map[string]any)
		}
		grouped[out.Name] = out.Value
	}
	if grouped != nil {
		scope.Set(alias, grouped)
	}
}
