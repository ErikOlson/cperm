// Package render is the adapter boundary between cperm's internal Policy
// (the source of truth) and concrete on-disk agent configuration formats.
//
// The rest of cperm composes, diffs, and reasons about a model.Policy and
// never touches a wire format directly. A Renderer is the only thing that
// knows the shape of a specific agent's settings file, so a future schema
// change — or support for a different agent — is confined to one
// implementation behind this interface.
package render

import "github.com/erikolson/cperm/internal/model"

// Renderer translates between a model.Policy and a concrete configuration
// wire format. One implementation exists today (ClaudeCode); the interface
// is the seam that keeps the core format-agnostic.
type Renderer interface {
	// Render serializes a Policy to the target wire format, including a
	// trailing newline, ready to write to disk or print.
	Render(p model.Policy) ([]byte, error)

	// Parse reads the target wire format back into a Policy. Only the
	// fields cperm manages are populated; unrecognized content is ignored.
	Parse(data []byte) (model.Policy, error)

	// OutputPath returns the file the rendered configuration is written to
	// for the given project directory.
	OutputPath(projectDir string) string

	// OverlayPaths returns additional files whose permissions layer on top of
	// the rendered output to form the effective on-disk state — e.g. Claude
	// Code's settings.local.json, which takes precedence and is where
	// interactive approvals are written. Callers skip files that don't exist.
	// Order is lowest-to-highest precedence.
	OverlayPaths(projectDir string) []string
}
