package importer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/erikolson/cperm/internal/model"
	"github.com/erikolson/cperm/internal/store"
)

// MatchResult describes how well an existing settings.json matches a module.
type MatchResult struct {
	ModuleName string
	Matched    []string // rules in the module that are present in the settings
	Unmatched  []string // rules in the module NOT present in the settings
	Total      int      // total rules in the module
	Coverage   float64  // Matched / Total
}

// ImportResult holds the complete analysis of an import operation.
type ImportResult struct {
	Matches        []MatchResult
	UnmatchedAllow []string // allow rules not covered by any module
	UnmatchedDeny  []string
	UnmatchedAsk   []string
	TotalRules     int
	CoveredRules   int
}

// Analyze compares a set of permissions (already parsed from an agent's
// settings by a render.Renderer) against all available modules.
func Analyze(perms model.Permissions, s *store.Store) (*ImportResult, error) {
	totalRules := len(perms.Allow) + len(perms.Deny) + len(perms.Ask)

	// Load all modules
	mods, err := s.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("loading modules: %w", err)
	}

	// Track which rules are covered by at least one module
	coveredAllow := make(map[string]bool)
	coveredDeny := make(map[string]bool)
	coveredAsk := make(map[string]bool)

	// Match each module against the settings
	var matches []MatchResult
	for _, mod := range mods {
		mr := matchModule(mod, &perms)
		if len(mr.Matched) > 0 {
			matches = append(matches, mr)
			for _, r := range mr.Matched {
				// Figure out which array this rule came from
				if contains(mod.Permissions.Allow, r) && contains(perms.Allow, r) {
					coveredAllow[r] = true
				}
				if contains(mod.Permissions.Deny, r) && contains(perms.Deny, r) {
					coveredDeny[r] = true
				}
				if contains(mod.Permissions.Ask, r) && contains(perms.Ask, r) {
					coveredAsk[r] = true
				}
			}
		}
	}

	// Sort matches by coverage descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Coverage > matches[j].Coverage
	})

	// Find unmatched rules
	unmatchedAllow := diff(perms.Allow, coveredAllow)
	unmatchedDeny := diff(perms.Deny, coveredDeny)
	unmatchedAsk := diff(perms.Ask, coveredAsk)

	coveredCount := len(coveredAllow) + len(coveredDeny) + len(coveredAsk)

	return &ImportResult{
		Matches:        matches,
		UnmatchedAllow: unmatchedAllow,
		UnmatchedDeny:  unmatchedDeny,
		UnmatchedAsk:   unmatchedAsk,
		TotalRules:     totalRules,
		CoveredRules:   coveredCount,
	}, nil
}

// SuggestModuleName generates a module name from a set of permission rules.
func SuggestModuleName(rules []string) string {
	// Look for common prefixes in Bash rules
	prefixes := make(map[string]int)
	for _, r := range rules {
		if strings.HasPrefix(r, "Bash(") {
			// Extract the command name: Bash(go:*) -> go
			inner := strings.TrimPrefix(r, "Bash(")
			inner = strings.TrimSuffix(inner, ")")
			if idx := strings.IndexAny(inner, ": "); idx > 0 {
				prefixes[inner[:idx]]++
			}
		}
		if strings.HasPrefix(r, "Read(") || strings.HasPrefix(r, "Edit(") {
			prefixes["files"]++
		}
		if strings.HasPrefix(r, "mcp__") {
			parts := strings.SplitN(r, "__", 3)
			if len(parts) >= 2 {
				prefixes["mcp-"+parts[1]]++
			}
		}
		if strings.HasPrefix(r, "WebFetch") || strings.HasPrefix(r, "WebSearch") {
			prefixes["web"]++
		}
	}

	if len(prefixes) == 0 {
		return "custom"
	}

	// Return the most common prefix
	var best string
	var bestCount int
	for p, c := range prefixes {
		if c > bestCount {
			best = p
			bestCount = c
		}
	}
	return best
}

// matchModule checks how well a module matches the given permissions.
func matchModule(mod *model.Module, perms *model.Permissions) MatchResult {
	var matched, unmatched []string

	// Check all rules in the module against the settings
	for _, r := range mod.Permissions.Allow {
		if contains(perms.Allow, r) {
			matched = append(matched, r)
		} else {
			unmatched = append(unmatched, r)
		}
	}
	for _, r := range mod.Permissions.Deny {
		if contains(perms.Deny, r) {
			matched = append(matched, r)
		} else {
			unmatched = append(unmatched, r)
		}
	}
	for _, r := range mod.Permissions.Ask {
		if contains(perms.Ask, r) {
			matched = append(matched, r)
		} else {
			unmatched = append(unmatched, r)
		}
	}

	total := len(mod.Permissions.Allow) + len(mod.Permissions.Deny) + len(mod.Permissions.Ask)
	coverage := 0.0
	if total > 0 {
		coverage = float64(len(matched)) / float64(total)
	}

	return MatchResult{
		ModuleName: mod.Name,
		Matched:    matched,
		Unmatched:  unmatched,
		Total:      total,
		Coverage:   coverage,
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func diff(all []string, covered map[string]bool) []string {
	var result []string
	for _, r := range all {
		if !covered[r] {
			result = append(result, r)
		}
	}
	return result
}
