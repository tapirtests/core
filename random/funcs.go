package random

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

const (
	lowerAlnum = "abcdefghijklmnopqrstuvwxyz0123456789"
	digits     = "0123456789"

	// maxLength limits string(n) and digits(n): longer values are almost
	// certainly a mistake and would bloat requests and reports.
	maxLength = 1024

	// emailPrefix makes generated accounts easy to find and clean up.
	// emailDomain is reserved for examples (RFC 2606): mail sent to it goes
	// nowhere, so tests never write to real people.
	emailPrefix = "test-"
	emailDomain = "example.com"
	emailLength = 10
)

// function describes a random.* function: how to check its arguments and how
// to produce a value. Arguments are the literals of a template call:
// json.Number or string.
type function struct {
	usage string // signature shown in messages, e.g. "int(min, max)"
	// check validates the arguments; it runs before a run (in the validator)
	// and again before every call.
	check func(args []any) error
	call  func(g *Generator, args []any) any
}

var functions = map[string]function{
	"uuid": {
		usage: "uuid",
		check: noArgs,
		call: func(g *Generator, _ []any) any {
			var b [16]byte
			binary.BigEndian.PutUint64(b[:8], g.rng.Uint64())
			binary.BigEndian.PutUint64(b[8:], g.rng.Uint64())
			b[6] = b[6]&0x0f | 0x40 // version 4
			b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
			s := hex.EncodeToString(b[:])
			return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
		},
	},
	"email": {
		usage: "email",
		check: noArgs,
		call: func(g *Generator, _ []any) any {
			return emailPrefix + g.chars(lowerAlnum, emailLength) + "@" + emailDomain
		},
	},
	"string": {
		usage: "string(n)",
		check: lengthArg,
		call: func(g *Generator, args []any) any {
			return g.chars(lowerAlnum, int(mustInt(args[0])))
		},
	},
	"digits": {
		usage: "digits(n)",
		check: lengthArg,
		call: func(g *Generator, args []any) any {
			return g.chars(digits, int(mustInt(args[0])))
		},
	},
	"int": {
		usage: "int(min, max)",
		check: func(args []any) error {
			if err := argCount(args, 2); err != nil {
				return err
			}
			min, err := intArg(args, 0, "min")
			if err != nil {
				return err
			}
			max, err := intArg(args, 1, "max")
			if err != nil {
				return err
			}
			if min > max {
				return fmt.Errorf("min %d is greater than max %d", min, max)
			}
			return nil
		},
		call: func(g *Generator, args []any) any {
			n := g.between(mustInt(args[0]), mustInt(args[1]))
			return json.Number(strconv.FormatInt(n, 10))
		},
	},
	"float": {
		usage: "float(min, max)",
		check: func(args []any) error {
			if err := argCount(args, 2); err != nil {
				return err
			}
			min, err := floatArg(args, 0, "min")
			if err != nil {
				return err
			}
			max, err := floatArg(args, 1, "max")
			if err != nil {
				return err
			}
			if min > max {
				return fmt.Errorf("min %v is greater than max %v", args[0], args[1])
			}
			if math.IsInf(max-min, 0) {
				return fmt.Errorf("the range from %v to %v is too wide", args[0], args[1])
			}
			return nil
		},
		call: func(g *Generator, args []any) any {
			min, max := mustFloat(args[0]), mustFloat(args[1])
			// Two decimals: enough for prices, and readable in reports.
			x := math.Round((min+g.rng.Float64()*(max-min))*100) / 100
			x = math.Min(math.Max(x, min), max) // rounding may step outside the range
			return json.Number(strconv.FormatFloat(x, 'f', -1, 64))
		},
	},
	"bool": {
		usage: "bool",
		check: noArgs,
		call: func(g *Generator, _ []any) any {
			return g.rng.IntN(2) == 1
		},
	},
	"pick": {
		usage: "pick(a, b, ...)",
		check: func(args []any) error {
			if len(args) == 0 {
				return fmt.Errorf("expects at least 1 argument, got 0")
			}
			return nil
		},
		call: func(g *Generator, args []any) any {
			return args[g.rng.IntN(len(args))]
		},
	},
}

// Names returns the names of all functions, sorted.
func Names() []string {
	names := make([]string, 0, len(functions))
	for name := range functions {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Check reports whether a call of the function with these arguments is
// valid, without generating anything. Arguments are literals, so a call can
// be checked before a run. args hold json.Number and string values.
func Check(name string, args []any) error {
	fn, ok := functions[name]
	if !ok {
		if s := suggest(name); s != "" {
			return fmt.Errorf("unknown function random.%s, did you mean random.%s?", name, s)
		}
		return fmt.Errorf("unknown function random.%s, available: %s", name, strings.Join(Names(), ", "))
	}
	if err := fn.check(args); err != nil {
		return fmt.Errorf("random.%s: %w", fn.usage, err)
	}
	return nil
}

// Call checks the call and returns the next value of the function from the
// stream. The result is a string, a json.Number or a bool; pick returns one
// of its arguments as is.
func (g *Generator) Call(name string, args []any) (any, error) {
	if err := Check(name, args); err != nil {
		return nil, err
	}
	return functions[name].call(g, args), nil
}

func noArgs(args []any) error { return argCount(args, 0) }

func argCount(args []any, want int) error {
	if len(args) != want {
		return fmt.Errorf("expects %d argument(s), got %d", want, len(args))
	}
	return nil
}

// lengthArg checks the single argument of string(n) and digits(n).
func lengthArg(args []any) error {
	if err := argCount(args, 1); err != nil {
		return err
	}
	n, err := intArg(args, 0, "n")
	if err != nil {
		return err
	}
	if n < 1 || n > maxLength {
		return fmt.Errorf("n must be between 1 and %d, got %d", maxLength, n)
	}
	return nil
}

func intArg(args []any, i int, name string) (int64, error) {
	num, ok := args[i].(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s must be an integer, got %s", name, describe(args[i]))
	}
	n, err := num.Int64()
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %s", name, num)
	}
	return n, nil
}

func floatArg(args []any, i int, name string) (float64, error) {
	num, ok := args[i].(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s must be a number, got %s", name, describe(args[i]))
	}
	f, err := num.Float64()
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, fmt.Errorf("%s must be a number, got %s", name, num)
	}
	return f, nil
}

// mustInt and mustFloat read arguments that check has already validated.
func mustInt(arg any) int64 {
	n, _ := arg.(json.Number).Int64()
	return n
}

func mustFloat(arg any) float64 {
	f, _ := arg.(json.Number).Float64()
	return f
}

func describe(arg any) string {
	if s, ok := arg.(string); ok {
		return strconv.Quote(s)
	}
	return fmt.Sprint(arg)
}

// suggest returns the function name closest to name, if it is close enough
// to be a likely typo.
func suggest(name string) string {
	best, bestDist := "", 3 // suggest only within 2 edits
	for _, candidate := range Names() {
		if d := editDistance(name, candidate); d < bestDist {
			best, bestDist = candidate, d
		}
	}
	return best
}

// editDistance is the Levenshtein distance between two short ASCII names.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
