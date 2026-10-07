package assert

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// equal reports whether two values are deeply equal the way JSON values
// are: numbers are compared by value, so 201 equals 201.0 whatever their Go
// types; strings, booleans and null by identity; arrays element by element
// in order; objects field by field. Values of different kinds are never
// equal: the string "57" is not the number 57.
func equal(a, b any) bool {
	if na, ok := toNumber(a); ok {
		nb, ok := toNumber(b)
		return ok && na.Cmp(nb) == 0
	}

	switch a := a.(type) {
	case nil:
		return b == nil
	case string:
		b, ok := b.(string)
		return ok && a == b
	case bool:
		b, ok := b.(bool)
		return ok && a == b
	case []any:
		b, ok := b.([]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !equal(a[i], b[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		b, ok := b.(map[string]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for key, av := range a {
			bv, ok := b[key]
			if !ok || !equal(av, bv) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// toNumber converts a number to an exact rational: json.Number as decoded
// from JSON, or a Go number as written in code. Exact arithmetic keeps large
// integers and decimal fractions from being distorted by float64.
func toNumber(v any) (*big.Rat, bool) {
	switch n := v.(type) {
	case json.Number:
		r, ok := new(big.Rat).SetString(n.String())
		return r, ok
	case int:
		return new(big.Rat).SetInt64(int64(n)), true
	case int8:
		return new(big.Rat).SetInt64(int64(n)), true
	case int16:
		return new(big.Rat).SetInt64(int64(n)), true
	case int32:
		return new(big.Rat).SetInt64(int64(n)), true
	case int64:
		return new(big.Rat).SetInt64(n), true
	case uint:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint8:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint16:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint32:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint64:
		return new(big.Rat).SetUint64(n), true
	case float32:
		r := new(big.Rat).SetFloat64(float64(n))
		return r, r != nil
	case float64:
		r := new(big.Rat).SetFloat64(n)
		return r, r != nil
	}
	return nil, false
}

// kindOf names the JSON kind of a value with an article, for messages.
func kindOf(v any) string {
	if _, ok := toNumber(v); ok {
		return "a number"
	}
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "a string"
	case bool:
		return "a boolean"
	case []any:
		return "an array"
	case map[string]any:
		return "an object"
	default:
		return fmt.Sprintf("a %T", v)
	}
}
