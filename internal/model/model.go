package model

// Module is the atomic unit of composable permissions.
// Stored as JSON files in the cperm store (~/.config/cperm/modules/).
type Module struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version,omitempty"`
	Requires    []string          `json:"requires,omitempty"`
	Permissions Permissions       `json:"permissions"`
	Env         map[string]string `json:"env,omitempty"`
}

// Permissions holds the three permission arrays that Claude Code understands.
type Permissions struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
	Ask   []string `json:"ask,omitempty"`
}

// ComposeFile is the per-project declaration (.claude/compose.json).
// It lists which modules to compose and optional project-specific overrides.
type ComposeFile struct {
	Modules  []string       `json:"modules"`
	Override *Permissions   `json:"override,omitempty"`
	Settings map[string]any `json:"settings,omitempty"`
}

// Policy is cperm's internal, format-neutral representation of a composed
// configuration. It is the source of truth; concrete wire formats such as
// Claude Code's .claude/settings.json are produced from a Policy by a
// render.Renderer, never the other way around. Policy intentionally carries
// no JSON tags — it is not a wire type.
type Policy struct {
	Permissions Permissions
	Env         map[string]string

	// Settings are passthrough top-level keys from the compose file
	// (e.g. defaultMode, sandbox) that a renderer emits verbatim.
	Settings map[string]any
}

// ComposedResult holds the output of a composition along with metadata
// useful for status/diff reporting.
type ComposedResult struct {
	Policy       Policy
	ModulesUsed  []string
	AllowCount   int
	DenyCount    int
	AskCount     int
	Deduplicated int
	Conflicts    []Conflict
}

// Conflict represents a permission rule that appears in conflicting arrays
// (e.g., the same rule in both allow and deny).
type Conflict struct {
	Rule    string
	InAllow bool
	InDeny  bool
	InAsk   bool
	Sources []string // which modules contributed this rule
}
