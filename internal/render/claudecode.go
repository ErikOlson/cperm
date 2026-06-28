package render

import (
	"encoding/json"
	"path/filepath"

	"github.com/erikolson/cperm/internal/model"
)

const (
	claudeDir      = ".claude"
	settingsOutput = "settings.json"
	settingsLocal  = "settings.local.json"
)

// ClaudeCode renders and parses Claude Code's .claude/settings.json format:
//
//	{
//	  "permissions": { "allow": [...], "ask": [...], "deny": [...] },
//	  "env": { ... },
//	  <passthrough top-level keys>
//	}
//
// It is the only type in cperm that knows this wire shape.
type ClaudeCode struct{}

// compile-time assurance that ClaudeCode satisfies the interface.
var _ Renderer = ClaudeCode{}

// OutputPath returns <projectDir>/.claude/settings.json.
func (ClaudeCode) OutputPath(projectDir string) string {
	return filepath.Join(projectDir, claudeDir, settingsOutput)
}

// OverlayPaths returns <projectDir>/.claude/settings.local.json — the local,
// gitignored file Claude Code writes interactive approvals into, which takes
// precedence over settings.json.
func (ClaudeCode) OverlayPaths(projectDir string) []string {
	return []string{filepath.Join(projectDir, claudeDir, settingsLocal)}
}

// Render serializes a Policy to settings.json bytes with a trailing newline.
// Empty permission arrays and an empty env are omitted; passthrough settings
// are written as top-level keys.
func (ClaudeCode) Render(p model.Policy) ([]byte, error) {
	out := make(map[string]any)

	perms := make(map[string]any)
	if len(p.Permissions.Allow) > 0 {
		perms["allow"] = p.Permissions.Allow
	}
	if len(p.Permissions.Deny) > 0 {
		perms["deny"] = p.Permissions.Deny
	}
	if len(p.Permissions.Ask) > 0 {
		perms["ask"] = p.Permissions.Ask
	}
	out["permissions"] = perms

	if len(p.Env) > 0 {
		out["env"] = p.Env
	}

	// Passthrough settings (e.g. defaultMode) at the top level.
	for k, v := range p.Settings {
		out[k] = v
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Parse reads settings.json bytes into a Policy. Only the permissions and env
// cperm manages are populated; other keys are ignored.
func (ClaudeCode) Parse(data []byte) (model.Policy, error) {
	var raw struct {
		Permissions model.Permissions `json:"permissions"`
		Env         map[string]string `json:"env"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return model.Policy{}, err
	}
	return model.Policy{Permissions: raw.Permissions, Env: raw.Env}, nil
}
