package model

// Value is a value written in tapir.json: an input, a default, an expected
// value. It holds what JSON decoding produces: nil, bool, json.Number, string,
// []any or map[string]any. Strings may contain templates such as
// "{{productId}}"; they are resolved by the engine, not by the model.
//
// In maps, a missing key and a key with a nil value mean different things:
// the value is not set versus an explicit null (for example, an intentional
// request without authorization).
type Value = any
