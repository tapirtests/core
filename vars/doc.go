// Package vars implements variables and templates: the "{{...}}" expressions
// written in tapir.json and the scopes they are resolved against.
//
// A template is a string with expressions in double braces:
//
//	"Bearer {{sellerToken}}"
//	"{{product.tags[0].title}}"
//	"{{env.BASE_URL}}/products/{{productId}}"
//	"{{random.int(1, 999)}}"
//
// Expressions are:
//
//   - a variable with an optional path into its value: name, a.b, items[0],
//     headers["Content-Type"];
//   - an env variable: env.NAME;
//   - a call of a random data function: random.email, random.int(1, 999).
//
// A backslash before the braces makes them literal: "\{{" is the text "{{".
// Closing braces need no escaping: outside an expression "}}" is plain text.
//
// The package knows nothing about HTTP or responses: values get into scopes
// from the engine, which extracts them from responses with JSONPath.
package vars
