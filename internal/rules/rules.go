// Package rules implements coverage (subsumption) semantics for Claude Code
// permission rules: does one rule already permit everything another would?
//
// This is a deliberately partial model of Claude Code's matching — it handles
// the cases that dominate real settings: bare tool names, Tool(*) / Tool(:*),
// and Bash/PowerShell command-prefix wildcards. It does not attempt full
// gitignore-style path subsumption for Read/Edit beyond exact and bare-tool
// matches, so it errs toward "not covered" (safe: a rule is only treated as
// redundant when coverage is certain).
package rules

import "strings"

// Covers reports whether rule a permits every use that rule b would — i.e. b is
// redundant whenever a is present.
func Covers(a, b string) bool {
	if a == b {
		return true
	}

	aTool, aSpec := split(a)
	bTool, bSpec := split(b)
	if aTool != bTool {
		return false
	}

	// A bare tool name (or Tool(*) / Tool(:*)) covers every use of that tool.
	if isWildcardSpec(aSpec) {
		return true
	}
	// b is the whole tool but a is scoped — a can't cover all of b.
	if bSpec == "" {
		return false
	}

	switch aTool {
	case "Bash", "PowerShell":
		aPrefix := commandPrefix(aSpec)
		bPrefix := commandPrefix(bSpec)
		if aPrefix == "" {
			return true
		}
		// Word-boundary prefix: "git" covers "git" and "git <more>", not "github".
		return bPrefix == aPrefix || strings.HasPrefix(bPrefix, aPrefix+" ")
	}
	return false
}

// CoveredBy reports whether candidate is covered by any rule in set.
func CoveredBy(candidate string, set []string) bool {
	for _, r := range set {
		if Covers(r, candidate) {
			return true
		}
	}
	return false
}

// split parses "Tool(spec)" into ("Tool", "spec"), or "Tool" into ("Tool", "").
func split(rule string) (tool, spec string) {
	open := strings.IndexByte(rule, '(')
	if open < 0 || !strings.HasSuffix(rule, ")") {
		return rule, ""
	}
	return rule[:open], rule[open+1 : len(rule)-1]
}

func isWildcardSpec(spec string) bool {
	return spec == "" || spec == "*" || spec == ":*"
}

// commandPrefix strips a trailing wildcard from a Bash specifier, yielding the
// literal command prefix: "git:*" -> "git", "git add *" -> "git add",
// "git add" -> "git add".
func commandPrefix(spec string) string {
	switch {
	case strings.HasSuffix(spec, ":*"):
		return strings.TrimSuffix(spec, ":*")
	case strings.HasSuffix(spec, " *"):
		return strings.TrimSuffix(spec, " *")
	case strings.HasSuffix(spec, "*"):
		return strings.TrimSuffix(spec, "*")
	default:
		return spec
	}
}
