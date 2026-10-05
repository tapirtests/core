package vars

// Scope is a storage of variables with a link to its parent.
//
// Scopes form a chain that mirrors the place of the running code in the
// project: the scope of a scenario is a child of the scope of its group,
// which is a child of the scope of the parent group, up to the root.
//
//	/root  ←  /root/products  ←  scenario productLifecycle
//
// Reading looks through the chain from the scope up to the root, so a
// scenario sees the variables of its group and of all parent groups.
// Writing always goes to the scope itself: a nested group or a scenario can
// shadow a variable of its parent but never changes it, and sibling groups
// do not see each other's variables.
//
// Env variables are not part of the chain: they live in their own namespace
// ({{env.NAME}}) and cannot be shadowed by a variable.
//
// A Scope is not safe for concurrent use. A parent may be read by several
// children at once only while nobody writes to it.
type Scope struct {
	parent *Scope
	vars   map[string]any
}

// NewScope returns an empty root scope.
func NewScope() *Scope {
	return &Scope{vars: make(map[string]any)}
}

// Child returns a new empty scope whose parent is s.
func (s *Scope) Child() *Scope {
	return &Scope{parent: s, vars: make(map[string]any)}
}

// Get returns the value of the variable, looking in s first and then up the
// chain. ok is false if no scope of the chain has the variable; a variable
// set to nil is found and returned as (nil, true).
func (s *Scope) Get(name string) (value any, ok bool) {
	for sc := s; sc != nil; sc = sc.parent {
		if value, ok = sc.vars[name]; ok {
			return value, true
		}
	}
	return nil, false
}

// Set stores the variable in s itself, never in a parent. If a parent has a
// variable with the same name, it is shadowed for s and its children and
// stays unchanged for everyone else.
func (s *Scope) Set(name string, value any) {
	s.vars[name] = value
}

// Local returns the names of the variables stored in s itself, without the
// parents, in no particular order.
func (s *Scope) Local() []string {
	names := make([]string, 0, len(s.vars))
	for name := range s.vars {
		names = append(names, name)
	}
	return names
}
