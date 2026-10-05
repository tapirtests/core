package vars_test

import (
	"slices"
	"testing"

	"github.com/tapirtests/core/vars"
)

// get fails the test if the variable is not found.
func get(t *testing.T, s *vars.Scope, name string) any {
	t.Helper()
	v, ok := s.Get(name)
	if !ok {
		t.Fatalf("variable %q not found", name)
	}
	return v
}

func TestScopeGetSet(t *testing.T) {
	s := vars.NewScope()
	if _, ok := s.Get("missing"); ok {
		t.Error("empty scope has a variable")
	}

	s.Set("token", "abc")
	if got := get(t, s, "token"); got != "abc" {
		t.Errorf("token = %v", got)
	}

	s.Set("token", "xyz")
	if got := get(t, s, "token"); got != "xyz" {
		t.Errorf("token after overwrite = %v", got)
	}
}

// A variable explicitly set to nil exists: "{{x}}" must resolve to null
// rather than fail with "unknown variable".
func TestScopeNilValueIsFound(t *testing.T) {
	s := vars.NewScope()
	s.Set("deletedAt", nil)

	v, ok := s.Get("deletedAt")
	if !ok {
		t.Fatal("variable set to nil is not found")
	}
	if v != nil {
		t.Errorf("value = %v, want nil", v)
	}
}

func TestScopeChildSeesParents(t *testing.T) {
	root := vars.NewScope()
	root.Set("sellerToken", "root-token")
	group := root.Child()
	group.Set("categoryId", 7)
	scenario := group.Child()

	if got := get(t, scenario, "sellerToken"); got != "root-token" {
		t.Errorf("variable of the grandparent = %v", got)
	}
	if got := get(t, scenario, "categoryId"); got != 7 {
		t.Errorf("variable of the parent = %v", got)
	}
}

func TestScopeParentDoesNotSeeChild(t *testing.T) {
	root := vars.NewScope()
	child := root.Child()
	child.Set("productId", 57)

	if _, ok := root.Get("productId"); ok {
		t.Error("parent sees a variable of its child")
	}
}

// Sibling groups are independent: this is what lets one of them fail, or
// lets them run in parallel later, without affecting the other.
func TestScopeSiblingsAreIsolated(t *testing.T) {
	root := vars.NewScope()
	products, orders := root.Child(), root.Child()
	products.Set("productId", 57)

	if _, ok := orders.Get("productId"); ok {
		t.Error("a scope sees a variable of its sibling")
	}
}

func TestScopeShadowing(t *testing.T) {
	root := vars.NewScope()
	root.Set("token", "root-token")
	child := root.Child()
	grandchild := child.Child()

	child.Set("token", "child-token")

	if got := get(t, child, "token"); got != "child-token" {
		t.Errorf("child sees %v, want its own value", got)
	}
	if got := get(t, grandchild, "token"); got != "child-token" {
		t.Errorf("grandchild sees %v, want the nearest value", got)
	}
	if got := get(t, root, "token"); got != "root-token" {
		t.Errorf("parent value changed to %v", got)
	}
	if got := get(t, root.Child(), "token"); got != "root-token" {
		t.Errorf("another child sees %v, want the parent value", got)
	}
}

// Shadowing with nil hides the parent value instead of falling through.
func TestScopeShadowingWithNil(t *testing.T) {
	root := vars.NewScope()
	root.Set("token", "root-token")
	child := root.Child()
	child.Set("token", nil)

	if got := get(t, child, "token"); got != nil {
		t.Errorf("child sees %v, want nil", got)
	}
}

// A child reads the parent live, not a snapshot taken at creation: a group
// scope keeps receiving outputs of scenarios while its children exist.
func TestScopeChildSeesLaterParentChanges(t *testing.T) {
	root := vars.NewScope()
	child := root.Child()

	root.Set("sellerToken", "abc")
	if got := get(t, child, "sellerToken"); got != "abc" {
		t.Errorf("child sees %v", got)
	}
}

func TestScopeLocal(t *testing.T) {
	root := vars.NewScope()
	root.Set("a", 1)
	child := root.Child()
	child.Set("b", 2)
	child.Set("c", 3)

	got := child.Local()
	slices.Sort(got)
	if !slices.Equal(got, []string{"b", "c"}) {
		t.Errorf("Local() = %v, want only the variables of the scope itself", got)
	}
	if got := root.Child().Local(); len(got) != 0 {
		t.Errorf("Local() of an empty scope = %v", got)
	}
}
