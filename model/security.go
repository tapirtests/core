package model

// SecurityScheme describes how the API accepts credentials, as declared in
// the API documentation (Swagger securityDefinitions).
//
// It holds only the placement rule, never a credential. A request that needs
// a scheme gets an "auth.<Name>" input; the value passed there is placed into
// the request according to this rule, e.g. "Authorization: Bearer <value>".
type SecurityScheme struct {
	Name      string       // bearer, ...
	Type      SecurityType // apiKey, basic, ...
	In        ParamIn      // header, query, ...
	ParamName string       // header or query parameter name, e.g. Authorization
	Prefix    string       // "Bearer "
}

// SecurityType is the kind of SecurityScheme.
type SecurityType string

const (
	// SecurityAPIKey places the credential as is (with an optional Prefix)
	// into a header or a query parameter. In Swagger 2.0 JWT/Bearer auth is
	// also described as apiKey in the Authorization header.
	SecurityAPIKey SecurityType = "apiKey"
)

// ParamIn is where a credential is placed in a request.
type ParamIn string

const (
	// InHeader places the credential into the header named ParamName.
	InHeader ParamIn = "header"
	// InQuery places the credential into the query parameter named ParamName.
	InQuery ParamIn = "query"
)
