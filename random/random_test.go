package random_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/tapirtests/core/random"
)

func num(s string) json.Number { return json.Number(s) }

// call fails the test on error and returns the value.
func call(t *testing.T, g *random.Generator, name string, args ...any) any {
	t.Helper()
	v, err := g.Call(name, args)
	if err != nil {
		t.Fatalf("random.%s%v: %v", name, args, err)
	}
	return v
}

// sequence draws a mix of values, enough to tell two streams apart.
func sequence(t *testing.T, g *random.Generator) []any {
	t.Helper()
	return []any{
		call(t, g, "uuid"),
		call(t, g, "email"),
		call(t, g, "string", num("8")),
		call(t, g, "int", num("1"), num("1000000")),
		call(t, g, "float", num("0"), num("1000")),
	}
}

func equal(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The values for a seed are a promise: a failed run is reproduced by its
// seed, possibly with a newer version of the core. If this test fails, the
// generation algorithm changed and old seeds no longer reproduce old data.
func TestGoldenValues(t *testing.T) {
	g := random.New(81723)
	golden := []struct {
		name string
		args []any
		want any
	}{
		{"uuid", nil, "70d16cf9-99e7-4b34-ab9a-10e3a5ed433e"},
		{"email", nil, "test-53nrbd26hi@example.com"},
		{"string", []any{num("8")}, "nf957vhp"},
		{"digits", []any{num("6")}, "444247"},
		{"int", []any{num("1"), num("999")}, num("162")},
		{"float", []any{num("1"), num("100")}, num("65.84")},
		{"bool", nil, true},
		{"pick", []any{"a", "b", "c"}, "c"},
	}
	for _, tt := range golden {
		if got := call(t, g, tt.name, tt.args...); got != tt.want {
			t.Errorf("random.%s%v = %#v, want %#v", tt.name, tt.args, got, tt.want)
		}
	}

	derived := random.New(81723).Derive("/root/products:productLifecycle")
	if got := call(t, derived, "email"); got != "test-h25j9wmqqq@example.com" {
		t.Errorf("derived email = %v", got)
	}
}

func TestSameSeedSameValues(t *testing.T) {
	a, b := sequence(t, random.New(42)), sequence(t, random.New(42))
	if !equal(a, b) {
		t.Errorf("same seed gives different values:\n %v\n %v", a, b)
	}
}

func TestDifferentSeedsDifferentValues(t *testing.T) {
	a, b := sequence(t, random.New(1)), sequence(t, random.New(2))
	if equal(a, b) {
		t.Errorf("different seeds give the same values: %v", a)
	}
}

func TestDerive(t *testing.T) {
	t.Run("same key gives the same stream", func(t *testing.T) {
		root := random.New(7)
		a, b := sequence(t, root.Derive("k")), sequence(t, root.Derive("k"))
		if !equal(a, b) {
			t.Errorf("streams differ:\n %v\n %v", a, b)
		}
	})

	t.Run("different keys give different streams", func(t *testing.T) {
		root := random.New(7)
		if equal(sequence(t, root.Derive("a")), sequence(t, root.Derive("b"))) {
			t.Error("streams of different keys are equal")
		}
	})

	t.Run("derived stream differs from its parent", func(t *testing.T) {
		if equal(sequence(t, random.New(7)), sequence(t, random.New(7).Derive("a"))) {
			t.Error("derived stream equals its parent")
		}
	})

	// The order in which scenarios run must not change their data.
	t.Run("does not depend on what the parent has produced", func(t *testing.T) {
		fresh := random.New(7)
		want := sequence(t, fresh.Derive("k"))

		used := random.New(7)
		sequence(t, used)
		sequence(t, used.Derive("other"))
		if got := sequence(t, used.Derive("k")); !equal(got, want) {
			t.Errorf("derived stream depends on the parent state:\n got  %v\n want %v", got, want)
		}
	})

	t.Run("does not advance the parent", func(t *testing.T) {
		want := sequence(t, random.New(7))

		g := random.New(7)
		sequence(t, g.Derive("a"))
		if got := sequence(t, g); !equal(got, want) {
			t.Errorf("Derive changed the parent stream:\n got  %v\n want %v", got, want)
		}
	})

	t.Run("nested keys are distinct", func(t *testing.T) {
		root := random.New(7)
		nested := sequence(t, root.Derive("a").Derive("b"))
		if equal(nested, sequence(t, root.Derive("b").Derive("a"))) {
			t.Error("a/b and b/a give the same stream")
		}
		if !equal(nested, sequence(t, root.Derive("a").Derive("b"))) {
			t.Error("nested derivation is not reproducible")
		}
	})

	t.Run("depends on the seed", func(t *testing.T) {
		if equal(sequence(t, random.New(1).Derive("k")), sequence(t, random.New(2).Derive("k"))) {
			t.Error("same key under different seeds gives the same stream")
		}
	})
}

var (
	uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	email  = regexp.MustCompile(`^test-[a-z0-9]{10}@example\.com$`)
	alnum  = regexp.MustCompile(`^[a-z0-9]+$`)
	digits = regexp.MustCompile(`^[0-9]+$`)
)

func TestValueFormats(t *testing.T) {
	g := random.New(3)
	for i := 0; i < 200; i++ {
		if v := call(t, g, "uuid").(string); !uuidV4.MatchString(v) {
			t.Fatalf("uuid %q is not a version 4 UUID", v)
		}
		if v := call(t, g, "email").(string); !email.MatchString(v) {
			t.Fatalf("email %q has an unexpected form", v)
		}
		if v := call(t, g, "string", num("12")).(string); len(v) != 12 || !alnum.MatchString(v) {
			t.Fatalf("string(12) = %q", v)
		}
		if v := call(t, g, "digits", num("7")).(string); len(v) != 7 || !digits.MatchString(v) {
			t.Fatalf("digits(7) = %q", v)
		}
	}
}

func TestValuesAreUnique(t *testing.T) {
	g := random.New(5)
	seen := make(map[any]bool)
	for i := 0; i < 5000; i++ {
		for _, name := range []string{"uuid", "email"} {
			v := call(t, g, name)
			if seen[v] {
				t.Fatalf("random.%s repeated %v within one stream", name, v)
			}
			seen[v] = true
		}
	}
}

func TestInt(t *testing.T) {
	g := random.New(11)

	t.Run("bounds are inclusive and all values appear", func(t *testing.T) {
		seen := make(map[int64]int)
		for i := 0; i < 600; i++ {
			n, err := call(t, g, "int", num("1"), num("3")).(json.Number).Int64()
			if err != nil {
				t.Fatal(err)
			}
			if n < 1 || n > 3 {
				t.Fatalf("int(1, 3) = %d", n)
			}
			seen[n]++
		}
		for n := int64(1); n <= 3; n++ {
			if seen[n] == 0 {
				t.Errorf("int(1, 3) never returned %d", n)
			}
		}
	})

	t.Run("negative range", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			n, _ := call(t, g, "int", num("-5"), num("-2")).(json.Number).Int64()
			if n < -5 || n > -2 {
				t.Fatalf("int(-5, -2) = %d", n)
			}
		}
	})

	t.Run("min equals max", func(t *testing.T) {
		if got := call(t, g, "int", num("7"), num("7")); got != num("7") {
			t.Errorf("int(7, 7) = %v", got)
		}
	})

	t.Run("whole int64 range", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			v := call(t, g, "int", num("-9223372036854775808"), num("9223372036854775807")).(json.Number)
			if _, err := v.Int64(); err != nil {
				t.Fatalf("result %v is not an int64: %v", v, err)
			}
		}
	})

	t.Run("range wider than int64 max", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			n, _ := call(t, g, "int", num("-9223372036854775808"), num("5")).(json.Number).Int64()
			if n > 5 {
				t.Fatalf("int(min int64, 5) = %d", n)
			}
		}
	})
}

func TestFloat(t *testing.T) {
	g := random.New(13)

	t.Run("within bounds with at most two decimals", func(t *testing.T) {
		for i := 0; i < 500; i++ {
			v := call(t, g, "float", num("1.5"), num("2.5")).(json.Number)
			f, err := v.Float64()
			if err != nil || f < 1.5 || f > 2.5 {
				t.Fatalf("float(1.5, 2.5) = %v (%v)", v, err)
			}
			if _, frac, ok := strings.Cut(v.String(), "."); ok && len(frac) > 2 {
				t.Fatalf("float(1.5, 2.5) = %v has more than two decimals", v)
			}
		}
	})

	t.Run("bounds with more than two decimals", func(t *testing.T) {
		for i := 0; i < 500; i++ {
			f, _ := call(t, g, "float", num("1.003"), num("1.004")).(json.Number).Float64()
			if f < 1.003 || f > 1.004 {
				t.Fatalf("float(1.003, 1.004) = %v is out of range", f)
			}
		}
	})

	t.Run("min equals max", func(t *testing.T) {
		if got := call(t, g, "float", num("2.5"), num("2.5")); got != num("2.5") {
			t.Errorf("float(2.5, 2.5) = %v", got)
		}
	})

	t.Run("integer bounds are accepted", func(t *testing.T) {
		f, _ := call(t, g, "float", num("0"), num("1")).(json.Number).Float64()
		if f < 0 || f > 1 {
			t.Errorf("float(0, 1) = %v", f)
		}
	})
}

func TestBool(t *testing.T) {
	g := random.New(17)
	seen := make(map[bool]bool)
	for i := 0; i < 200; i++ {
		seen[call(t, g, "bool").(bool)] = true
	}
	if !seen[true] || !seen[false] {
		t.Errorf("bool returned only %v", seen)
	}
}

func TestPick(t *testing.T) {
	g := random.New(19)

	t.Run("returns every argument, with its type", func(t *testing.T) {
		args := []any{"draft", num("2"), "sold"}
		seen := make(map[any]bool)
		for i := 0; i < 300; i++ {
			seen[call(t, g, "pick", args...)] = true
		}
		for _, arg := range args {
			if !seen[arg] {
				t.Errorf("pick never returned %#v", arg)
			}
		}
		if len(seen) != len(args) {
			t.Errorf("pick returned values that are not arguments: %v", seen)
		}
	})

	t.Run("single argument", func(t *testing.T) {
		if got := call(t, g, "pick", "only"); got != "only" {
			t.Errorf("pick(only) = %v", got)
		}
	})
}

func TestNames(t *testing.T) {
	want := []string{"bool", "digits", "email", "float", "int", "pick", "string", "uuid"}
	got := random.Names()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		fn      string
		args    []any
		wantErr string // substring of the error; empty means the call is valid
	}{
		// Valid calls.
		{"uuid", "uuid", nil, ""},
		{"email", "email", nil, ""},
		{"string", "string", []any{num("8")}, ""},
		{"string max length", "string", []any{num("1024")}, ""},
		{"digits", "digits", []any{num("1")}, ""},
		{"int", "int", []any{num("-5"), num("5")}, ""},
		{"float", "float", []any{num("0.5"), num("10")}, ""},
		{"bool", "bool", nil, ""},
		{"pick", "pick", []any{"a", num("1")}, ""},

		// Unknown functions.
		{"typo", "emial", nil, "did you mean random.email?"},
		{"typo in case", "Int", nil, "did you mean random.int?"},
		{"unknown", "address", nil, "available: bool, digits, email"},

		// Argument count.
		{"uuid with argument", "uuid", []any{num("1")}, "expects 0 argument(s), got 1"},
		{"string without argument", "string", nil, "expects 1 argument(s), got 0"},
		{"int with one argument", "int", []any{num("1")}, "expects 2 argument(s), got 1"},
		{"int with three arguments", "int", []any{num("1"), num("2"), num("3")}, "expects 2 argument(s), got 3"},
		{"pick without arguments", "pick", nil, "at least 1 argument"},

		// Argument types.
		{"string with a string", "string", []any{"8"}, `n must be an integer, got "8"`},
		{"string with a float", "string", []any{num("1.5")}, "n must be an integer, got 1.5"},
		{"int with a string", "int", []any{"a", num("2")}, `min must be an integer, got "a"`},
		{"int with a float", "int", []any{num("1"), num("2.5")}, "max must be an integer, got 2.5"},
		{"int beyond int64", "int", []any{num("1"), num("99999999999999999999")}, "max must be an integer"},
		{"float with a string", "float", []any{num("1"), "x"}, `max must be a number, got "x"`},

		// Argument values.
		{"string of zero length", "string", []any{num("0")}, "n must be between 1 and 1024, got 0"},
		{"string of negative length", "string", []any{num("-3")}, "n must be between 1 and 1024, got -3"},
		{"string too long", "string", []any{num("1025")}, "n must be between 1 and 1024, got 1025"},
		{"digits of zero length", "digits", []any{num("0")}, "n must be between 1 and 1024"},
		{"int min greater than max", "int", []any{num("10"), num("1")}, "min 10 is greater than max 1"},
		{"float min greater than max", "float", []any{num("2.5"), num("1")}, "min 2.5 is greater than max 1"},
		{"float range too wide", "float", []any{num("-1e308"), num("1e308")}, "too wide"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := random.Check(tt.fn, tt.args)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("Check: unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("Check: expected an error containing %q", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("Check: error %q does not contain %q", err, tt.wantErr)
			}

			// Call applies the same checks.
			_, callErr := random.New(1).Call(tt.fn, tt.args)
			if (callErr == nil) != (err == nil) {
				t.Errorf("Call error = %v, Check error = %v", callErr, err)
			}
		})
	}
}

// An invalid call must not consume values: otherwise a broken step would
// shift the data of the steps after it.
func TestInvalidCallDoesNotAdvanceTheStream(t *testing.T) {
	want := sequence(t, random.New(23))

	g := random.New(23)
	if _, err := g.Call("int", []any{num("10"), num("1")}); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := g.Call("nope", nil); err == nil {
		t.Fatal("expected an error")
	}
	if got := sequence(t, g); !equal(got, want) {
		t.Errorf("stream advanced after invalid calls:\n got  %v\n want %v", got, want)
	}
}
