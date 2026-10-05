package vars_test

import (
	"reflect"
	"testing"

	"github.com/tapirtests/core/vars"
)

func TestScan(t *testing.T) {
	value := map[string]any{
		"plain":  "no templates here",
		"number": num("5"),
		"name":   "{{product.name}}",
		"url":    "{{env.BASE_URL}}/p/{{productId}}",
		"broken": "{{token",
		"items": []any{
			"static",
			map[string]any{"owner-id": "{{random.int(1, 9)}}"},
		},
		"escaped": `\{{not a template}}`,
	}

	type found struct {
		at, src string
		exprs   []string
		err     string
	}
	var got []found
	for _, occ := range vars.Scan(value) {
		f := found{at: occ.At(), src: occ.Src}
		if occ.Err != nil {
			f.err = occ.Err.Msg
		} else {
			for _, e := range occ.Template.Exprs() {
				f.exprs = append(f.exprs, e.String())
			}
		}
		got = append(got, f)
	}

	// Sorted by key; plain text, numbers and escaped braces are skipped.
	want := []found{
		{at: ".broken", src: "{{token", err: `unclosed "{{"`},
		{at: `.items[1]["owner-id"]`, src: "{{random.int(1, 9)}}", exprs: []string{"random.int(1, 9)"}},
		{at: ".name", src: "{{product.name}}", exprs: []string{"product.name"}},
		{at: ".url", src: "{{env.BASE_URL}}/p/{{productId}}", exprs: []string{"env.BASE_URL", "productId"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan\n got  %+v\n want %+v", got, want)
	}
}

func TestScanTopLevelString(t *testing.T) {
	occs := vars.Scan("Bearer {{token}}")
	if len(occs) != 1 || occs[0].At() != "" || len(occs[0].Path) != 0 {
		t.Fatalf("Scan = %+v", occs)
	}
	if occs := vars.Scan("plain"); len(occs) != 0 {
		t.Errorf("Scan(plain) = %+v, want nothing", occs)
	}
	if occs := vars.Scan(nil); len(occs) != 0 {
		t.Errorf("Scan(nil) = %+v, want nothing", occs)
	}
}

// Paths of occurrences must not share memory: each is built while walking.
func TestScanPathsAreIndependent(t *testing.T) {
	occs := vars.Scan(map[string]any{
		"a": []any{"{{x}}", "{{y}}", "{{z}}"},
	})
	if len(occs) != 3 {
		t.Fatalf("got %d occurrences", len(occs))
	}
	for i, occ := range occs {
		want := []vars.Segment{{Field: "a"}, {Index: i, IsIndex: true}}
		if !reflect.DeepEqual(occ.Path, want) {
			t.Errorf("occurrence %d: path = %+v, want %+v", i, occ.Path, want)
		}
	}
}

func TestVarAndEnvNames(t *testing.T) {
	value := map[string]any{
		"auth.bearer": "{{sellerToken}}",
		"path.id":     "{{productLifecycle.productId}}",
		"body": map[string]any{
			"name":  "{{product.tags[0]}} by {{sellerToken}}",
			"url":   "{{env.BASE_URL}}/{{env.API_PREFIX}}/{{env.BASE_URL}}",
			"email": "{{random.email}}",
			"bad":   "{{broken",
		},
	}

	// Root names only, sorted, without duplicates; random calls and broken
	// templates are not variables.
	if got, want := vars.VarNames(value), []string{"product", "productLifecycle", "sellerToken"}; !reflect.DeepEqual(got, want) {
		t.Errorf("VarNames = %v, want %v", got, want)
	}
	if got, want := vars.EnvNames(value), []string{"API_PREFIX", "BASE_URL"}; !reflect.DeepEqual(got, want) {
		t.Errorf("EnvNames = %v, want %v", got, want)
	}
	if got := vars.VarNames("plain"); len(got) != 0 {
		t.Errorf("VarNames(plain) = %v", got)
	}
}

func TestIsReserved(t *testing.T) {
	for name, want := range map[string]bool{
		"env": true, "random": true,
		"Env": false, "environment": false, "randomId": false, "token": false, "": false,
	} {
		if got := vars.IsReserved(name); got != want {
			t.Errorf("IsReserved(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestStoreOutputs(t *testing.T) {
	outputs := []vars.Output{{Name: "token", Value: "abc"}, {Name: "userId", Value: num("7")}}

	t.Run("outputs are grouped under the alias by default", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "login", outputs, nil)

		r := vars.Resolver{Scope: s}
		if got, err := r.Resolve("{{login.token}}/{{login.userId}}"); err != nil || got != "abc/7" {
			t.Errorf("Resolve = %v, %v", got, err)
		}
		if _, ok := s.Get("token"); ok {
			t.Error("output is also stored as a plain variable")
		}
	})

	t.Run("renamed output becomes a plain variable", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "login", outputs, map[string]string{"token": "sellerToken"})

		if got := get(t, s, "sellerToken"); got != "abc" {
			t.Errorf("sellerToken = %v", got)
		}
		// The rest stays under the alias; the renamed one does not.
		want := map[string]any{"userId": num("7")}
		if got := get(t, s, "login"); !reflect.DeepEqual(got, want) {
			t.Errorf("login = %#v, want %#v", got, want)
		}
	})

	t.Run("no alias object when everything is renamed", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "login", outputs, map[string]string{"token": "t", "userId": "u"})
		if _, ok := s.Get("login"); ok {
			t.Error("an empty alias object is stored")
		}
	})

	t.Run("no outputs store nothing", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "healthCheck", nil, nil)
		if got := s.Local(); len(got) != 0 {
			t.Errorf("stored %v", got)
		}
	})

	t.Run("two calls of one scenario do not overwrite each other", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "sellerLogin", []vars.Output{{Name: "token", Value: "seller"}}, nil)
		vars.StoreOutputs(s, "userLogin", []vars.Output{{Name: "token", Value: "user"}}, nil)

		r := vars.Resolver{Scope: s}
		if got, err := r.Resolve("{{sellerLogin.token}}/{{userLogin.token}}"); err != nil || got != "seller/user" {
			t.Errorf("Resolve = %v, %v", got, err)
		}
	})

	t.Run("rename overwrites an existing variable", func(t *testing.T) {
		s := vars.NewScope()
		s.Set("token", "old")
		vars.StoreOutputs(s, "login", []vars.Output{{Name: "token", Value: "new"}}, map[string]string{"token": "token"})
		if got := get(t, s, "token"); got != "new" {
			t.Errorf("token = %v, want the last write", got)
		}
	})

	t.Run("last output wins when two are renamed to one name", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "login",
			[]vars.Output{{Name: "a", Value: "first"}, {Name: "b", Value: "second"}},
			map[string]string{"a": "x", "b": "x"})
		if got := get(t, s, "x"); got != "second" {
			t.Errorf("x = %v, want the output declared last", got)
		}
	})

	t.Run("outputs go to the given scope only", func(t *testing.T) {
		parent := vars.NewScope()
		child := parent.Child()
		vars.StoreOutputs(child, "login", outputs, map[string]string{"token": "sellerToken"})
		if got := parent.Local(); len(got) != 0 {
			t.Errorf("parent received %v", got)
		}
	})

	t.Run("null output is stored", func(t *testing.T) {
		s := vars.NewScope()
		vars.StoreOutputs(s, "get", []vars.Output{{Name: "deletedAt", Value: nil}}, nil)
		r := vars.Resolver{Scope: s}
		if got, err := r.Resolve("{{get.deletedAt}}"); err != nil || got != nil {
			t.Errorf("Resolve = %v, %v", got, err)
		}
	})
}
