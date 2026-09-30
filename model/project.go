package model

// Project is the root of the model: everything described by one tapir.json
// together with the operations imported from the API specification.
// It holds the four global objects: the Requests, Scenario and Env variable
// buckets and the root group.
type Project struct {
	TapirVersion string // version of the tapir.json format (not of the core), e.g. "1"
	Name         string // project name
	BaseURL      string // API base URL; taken from the specification on import, may be a template like "{{env.BASE_URL}}"

	Spec            SpecRef                    // API specification the requests were imported from
	SecuritySchemes map[string]*SecurityScheme // how the API accepts credentials, keyed by scheme name: bearer, ...

	Env       []EnvVarDecl                // env variables bucket
	Requests  map[RequestID]*RequestDef   // requests bucket
	Scenarios map[ScenarioID]*ScenarioDef // scenarios bucket
	Root      *Group                      // root group
}

// SpecRef points to the snapshot of the API specification the project was
// built from. The hash lets Tapir detect that the specification changed and
// show a diff of affected steps.
type SpecRef struct {
	Type string // specification format: swagger2, later openapi3, ...
	Path string // path to the specification snapshot, relative to tapir.json
	Hash string // hash of the snapshot, "sha256:..."
}
