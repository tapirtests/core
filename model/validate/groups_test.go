package validate_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tapirtests/core/diag"
	"github.com/tapirtests/core/model"
	"github.com/tapirtests/core/model/validate"
)

// products returns the /root/products group of the reference project:
//
//	main:     productLifecycle, createProductUnauthorized
//	tearDown: cleanupProduct
func products(p *model.Project) *model.Group {
	return p.Root.Main.Groups[0]
}

// Checks of the group tree and of scenario calls; see groups.go.
// The reference tree is:
//
//	/root
//	  setup: login (alias sellerLogin), login (alias userLogin)
//	  main:  /root/products
func TestGroups(t *testing.T) {
	const productsPtr = "/root/main/groups/0"

	runBreakCases(t, []breakCase{
		// Valid variations.
		{
			name: "group name with dash and digits",
			breaks: func(p *model.Project) {
				products(p).Name = "products-v2"
			},
			want: nil,
		},
		{
			name: "same group name under different parents",
			breaks: func(p *model.Project) {
				products(p).Main.Groups = []*model.Group{{Name: "products"}}
			},
			want: nil,
		},
		{
			name: "same alias in different groups",
			breaks: func(p *model.Project) {
				products(p).Setup = []model.ScenarioCall{{
					Alias:      "sellerLogin",
					ScenarioID: "login",
					Inputs:     map[string]model.Value{"username": "u", "password": "p"},
				}}
			},
			want: nil,
		},
		{
			name: "group variables",
			breaks: func(p *model.Project) {
				products(p).Vars = map[string]model.Value{"currency": "EUR", "_limit": json.Number("10")}
			},
			want: nil,
		},

		// Tree.
		{
			name: "nil nested group",
			breaks: func(p *model.Project) {
				p.Root.Main.Groups = append(p.Root.Main.Groups, nil)
			},
			want: []want{{validate.V0802, diag.Error, "/root/main/groups/1"}},
		},
		{
			name:   "missing group name",
			breaks: func(p *model.Project) { products(p).Name = "" },
			want:   []want{{validate.V0803, diag.Error, productsPtr + "/name"}},
		},
		{
			name:   "group name with a slash",
			breaks: func(p *model.Project) { products(p).Name = "shop/products" },
			want:   []want{{validate.V0804, diag.Error, productsPtr + "/name"}},
		},
		{
			name:   "group name with a space",
			breaks: func(p *model.Project) { products(p).Name = "my products" },
			want:   []want{{validate.V0804, diag.Error, productsPtr + "/name"}},
		},
		{
			name: "duplicate sibling group name is reported on the repeat only",
			breaks: func(p *model.Project) {
				p.Root.Main.Groups = append(p.Root.Main.Groups, &model.Group{Name: "products"})
			},
			want: []want{{validate.V0805, diag.Error, "/root/main/groups/1/name"}},
		},
		{
			// The same pointer twice: the group would run twice and share
			// its scope. Its name is not reported as a duplicate on top.
			name: "group placed twice",
			breaks: func(p *model.Project) {
				p.Root.Main.Groups = append(p.Root.Main.Groups, products(p))
			},
			want: []want{{validate.V0806, diag.Error, "/root/main/groups/1"}},
		},
		{
			name: "group nested into itself",
			breaks: func(p *model.Project) {
				g := products(p)
				g.Main.Groups = append(g.Main.Groups, g)
			},
			want: []want{{validate.V0806, diag.Error, productsPtr + "/main/groups/0"}},
		},
		{
			name: "root nested into its descendant",
			breaks: func(p *model.Project) {
				g := products(p)
				g.Main.Groups = append(g.Main.Groups, &model.Group{
					Name: "deep",
					Main: model.Main{Groups: []*model.Group{p.Root}},
				})
			},
			want: []want{{validate.V0806, diag.Error, productsPtr + "/main/groups/0/main/groups/0"}},
		},
		{
			name: "invalid group variable name",
			breaks: func(p *model.Project) {
				products(p).Vars = map[string]model.Value{"base-price": json.Number("1")}
			},
			want: []want{{validate.V0807, diag.Error, productsPtr + "/vars/base-price"}},
		},
		{
			name: "empty group variable name",
			breaks: func(p *model.Project) {
				p.Root.Vars = map[string]model.Value{"": "x"}
			},
			want: []want{{validate.V0807, diag.Error, "/root/vars/"}},
		},

		// Scenario reference.
		{
			name:   "missing scenario ID",
			breaks: func(p *model.Project) { p.Root.Setup[0].ScenarioID = "" },
			// The alias is set, so it still takes part in the uniqueness check.
			want: []want{{validate.V0808, diag.Error, "/root/setup/0/scenarioId"}},
		},
		{
			name: "missing scenario ID and alias",
			breaks: func(p *model.Project) {
				products(p).Main.Scenarios[1] = model.ScenarioCall{}
			},
			want: []want{{validate.V0808, diag.Error, productsPtr + "/main/scenarios/1/scenarioId"}},
		},
		{
			// Inputs are not checked against a scenario that does not exist.
			name: "unknown scenario does not cascade to inputs",
			breaks: func(p *model.Project) {
				p.Root.Setup[0].ScenarioID = "logn"
			},
			want: []want{{validate.V0809, diag.Error, "/root/setup/0/scenarioId"}},
		},
		{
			// The nil entry is reported once, with scenarios.
			name:   "calls to a nil scenario are not checked",
			breaks: func(p *model.Project) { p.Scenarios["login"] = nil },
			want:   []want{{validate.V0700, diag.Error, "/scenarios/login"}},
		},

		// Aliases.
		{
			name:   "alias with a dot",
			breaks: func(p *model.Project) { p.Root.Setup[0].Alias = "seller.login" },
			want:   []want{{validate.V0810, diag.Error, "/root/setup/0/alias"}},
		},
		{
			name:   "duplicate explicit alias",
			breaks: func(p *model.Project) { p.Root.Setup[1].Alias = "sellerLogin" },
			want:   []want{{validate.V0811, diag.Error, "/root/setup/1/alias"}},
		},
		{
			name:   "implicit alias next to explicit ones",
			breaks: func(p *model.Project) { p.Root.Setup[1].Alias = "" },
			want:   nil, // implicit "login" differs from "sellerLogin"
		},
		{
			name: "same scenario called twice without aliases",
			breaks: func(p *model.Project) {
				g := products(p)
				g.Main.Scenarios = append(g.Main.Scenarios, model.ScenarioCall{ScenarioID: "productLifecycle"})
			},
			want: []want{{validate.V0811, diag.Error, productsPtr + "/main/scenarios/2/alias"}},
		},
		{
			name: "aliases are unique across group sections",
			breaks: func(p *model.Project) {
				g := products(p)
				g.TearDown = append(g.TearDown, model.ScenarioCall{ScenarioID: "productLifecycle"})
			},
			want: []want{{validate.V0811, diag.Error, productsPtr + "/tearDown/1/alias"}},
		},
		{
			name: "explicit alias equal to an implicit one",
			breaks: func(p *model.Project) {
				g := products(p)
				g.TearDown[0].Alias = "productLifecycle"
			},
			want: []want{{validate.V0811, diag.Error, productsPtr + "/tearDown/0/alias"}},
		},

		// Inputs and outputs.
		{
			name: "unknown scenario input",
			breaks: func(p *model.Project) {
				p.Root.Setup[0].Inputs["login"] = "x"
			},
			want: []want{{validate.V0812, diag.Error, "/root/setup/0/inputs/login"}},
		},
		{
			// Required inputs may come from the group scope: not checked here.
			name: "required input not passed explicitly",
			breaks: func(p *model.Project) {
				delete(p.Root.Setup[0].Inputs, "password")
			},
			want: nil,
		},
		{
			name: "rename of an unknown output",
			breaks: func(p *model.Project) {
				p.Root.Setup[0].Outputs["jwt"] = "sellerJwt"
			},
			want: []want{{validate.V0813, diag.Error, "/root/setup/0/outputs/jwt"}},
		},
		{
			name: "rename to an invalid name",
			breaks: func(p *model.Project) {
				p.Root.Setup[0].Outputs["token"] = "seller.token"
			},
			want: []want{{validate.V0814, diag.Error, "/root/setup/0/outputs/token"}},
		},
		{
			name: "rename of an unknown output to an invalid name",
			breaks: func(p *model.Project) {
				p.Root.Setup[0].Outputs["jwt"] = ""
			},
			want: []want{
				{validate.V0813, diag.Error, "/root/setup/0/outputs/jwt"},
				{validate.V0814, diag.Error, "/root/setup/0/outputs/jwt"},
			},
		},

		// Problems deep in the tree are found.
		{
			name: "problems in a nested group",
			breaks: func(p *model.Project) {
				products(p).Main.Groups = []*model.Group{{
					Name: "deep",
					Main: model.Main{
						Groups: []*model.Group{{
							Name:  "deeper",
							Setup: []model.ScenarioCall{{ScenarioID: "nope"}},
						}},
					},
				}}
			},
			want: []want{{validate.V0809, diag.Error, productsPtr + "/main/groups/0/main/groups/0/setup/0/scenarioId"}},
		},
	})
}

// A cycle in the tree must not hang the validator.
func TestGroupCycleTerminates(t *testing.T) {
	p := &model.Project{FormatVersion: "1", BaseURL: "x", Name: "n", Root: &model.Group{Name: "root"}}
	a := &model.Group{Name: "a"}
	b := &model.Group{Name: "b", Main: model.Main{Groups: []*model.Group{a}}}
	a.Main.Groups = []*model.Group{b}
	p.Root.Main.Groups = []*model.Group{a}

	done := make(chan diag.List, 1)
	go func() { done <- validate.Project(p, testFile) }()

	select {
	case got := <-done:
		checkDiags(t, got, []want{{validate.V0806, diag.Error, "/root/main/groups/0/main/groups/0/main/groups/0"}})
	case <-time.After(5 * time.Second):
		t.Fatal("validator did not finish: cycle in the group tree is not detected")
	}
}
