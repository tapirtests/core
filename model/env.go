package model

// EnvVarDecl declares a variable of the Env variable bucket, available in
// templates as {{env.NAME}}.
//
// The bucket is read-only during a run. Secrets and plain settings (base URL,
// logins) are kept in the same bucket and differ only by the Secret flag.
type EnvVarDecl struct {
	Name        string // variable name, e.g. SELLER_PASS
	Secret      bool   // mask the value in logs and reports
	Description string // human-readable purpose of the variable
}
