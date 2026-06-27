package composer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/store"
)

const (
	composeFileName = "compose.json"
	claudeDir       = ".claude"
)

// Composer handles loading compose files and producing merged settings.
type Composer struct {
	Store *store.Store
}

// New creates a composer backed by the given store.
func New(s *store.Store) *Composer {
	return &Composer{Store: s}
}

// ComposeFilePath returns the expected path to the compose file in the project.
func ComposeFilePath(projectDir string) string {
	return filepath.Join(projectDir, claudeDir, composeFileName)
}

// LoadComposeFile reads and parses a compose.json.
func LoadComposeFile(path string) (*model.ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf model.ComposeFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}
	return &cf, nil
}

// SaveComposeFile writes a compose.json to disk.
func SaveComposeFile(path string, cf *model.ComposeFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// Compose resolves modules, merges permissions, and returns the result.
func (c *Composer) Compose(cf *model.ComposeFile) (*model.ComposedResult, error) {
	// Resolve module order with dependency expansion
	resolved, err := c.resolveModules(cf.Modules)
	if err != nil {
		return nil, err
	}

	// Merge permissions from all modules
	var allAllow, allDeny, allAsk []string
	env := make(map[string]string)
	sources := make(map[string][]string) // rule -> which modules contributed it

	for _, name := range resolved {
		mod, err := c.Store.Load(name)
		if err != nil {
			return nil, fmt.Errorf("loading module %q: %w", name, err)
		}

		for _, r := range mod.Permissions.Allow {
			allAllow = append(allAllow, r)
			sources[r] = append(sources[r], name)
		}
		for _, r := range mod.Permissions.Deny {
			allDeny = append(allDeny, r)
			sources[r] = append(sources[r], name)
		}
		for _, r := range mod.Permissions.Ask {
			allAsk = append(allAsk, r)
			sources[r] = append(sources[r], name)
		}
		for k, v := range mod.Env {
			env[k] = v
		}
	}

	// Apply overrides from compose file
	if cf.Override != nil {
		for _, r := range cf.Override.Allow {
			allAllow = append(allAllow, r)
			sources[r] = append(sources[r], "<override>")
		}
		for _, r := range cf.Override.Deny {
			allDeny = append(allDeny, r)
			sources[r] = append(sources[r], "<override>")
		}
		for _, r := range cf.Override.Ask {
			allAsk = append(allAsk, r)
			sources[r] = append(sources[r], "<override>")
		}
	}

	// Deduplicate
	beforeCount := len(allAllow) + len(allDeny) + len(allAsk)
	allAllow = uniqueStrings(allAllow)
	allDeny = uniqueStrings(allDeny)
	allAsk = uniqueStrings(allAsk)
	afterCount := len(allAllow) + len(allDeny) + len(allAsk)

	// Detect conflicts (same rule in multiple arrays)
	conflicts := detectConflicts(allAllow, allDeny, allAsk, sources)

	// Build the composed policy (format-neutral source of truth).
	policy := model.Policy{
		Permissions: model.Permissions{
			Allow: allAllow,
			Deny:  allDeny,
			Ask:   allAsk,
		},
		Env:      env,
		Settings: cf.Settings,
	}

	return &model.ComposedResult{
		Policy:       policy,
		ModulesUsed:  resolved,
		AllowCount:   len(allAllow),
		DenyCount:    len(allDeny),
		AskCount:     len(allAsk),
		Deduplicated: beforeCount - afterCount,
		Conflicts:    conflicts,
	}, nil
}

// resolveModules expands module dependencies via topological sort.
// For v0.1, this does a simple depth-first expansion with cycle detection.
func (c *Composer) resolveModules(requested []string) ([]string, error) {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var order []string

	var visit func(name string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if visited[name] {
			return nil
		}

		inStack[name] = true
		mod, err := c.Store.Load(name)
		if err != nil {
			return err
		}

		// Resolve dependencies first
		for _, dep := range mod.Requires {
			if err := visit(dep); err != nil {
				return err
			}
		}

		inStack[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	for _, name := range requested {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// uniqueStrings deduplicates a string slice preserving order.
func uniqueStrings(s []string) []string {
	seen := make(map[string]bool, len(s))
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// detectConflicts finds rules that appear in multiple permission arrays.
func detectConflicts(allow, deny, ask []string, sources map[string][]string) []model.Conflict {
	allowSet := toSet(allow)
	denySet := toSet(deny)
	askSet := toSet(ask)

	allRules := make(map[string]bool)
	for _, r := range allow {
		allRules[r] = true
	}
	for _, r := range deny {
		allRules[r] = true
	}
	for _, r := range ask {
		allRules[r] = true
	}

	var conflicts []model.Conflict
	for rule := range allRules {
		inAllow := allowSet[rule]
		inDeny := denySet[rule]
		inAsk := askSet[rule]

		count := 0
		if inAllow {
			count++
		}
		if inDeny {
			count++
		}
		if inAsk {
			count++
		}

		if count > 1 {
			conflicts = append(conflicts, model.Conflict{
				Rule:    rule,
				InAllow: inAllow,
				InDeny:  inDeny,
				InAsk:   inAsk,
				Sources: sources[rule],
			})
		}
	}
	return conflicts
}

func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}
