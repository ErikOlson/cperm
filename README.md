# cperm

[![CI](https://github.com/ErikOlson/cperm/actions/workflows/ci.yml/badge.svg)](https://github.com/ErikOlson/cperm/actions/workflows/ci.yml)

**Composable Claude Code permissions.**

Claude Code's permission system is powerful but manual. You approve commands one by one, they accumulate in `settings.json`, and eventually every project has a slightly different, organically-grown permissions file that you can't reproduce, can't share, and can't tell if it's drifted.

`cperm` fixes this with a Nix-inspired approach: define reusable **permission modules**, compose them per-project, and get deterministic, reproducible `settings.json` files.

## The Problem

```
# Project A's settings.json: 47 rules, accumulated over months
# Project B's settings.json: 31 rules, mostly the same but subtly different
# Project C: you forgot to set up permissions, running naked
# Your coworker: completely different set of rules
```

## The Solution

```bash
# Define your toolchain once
cperm new rust  # interactive — enter your Rust permission rules

# Compose per-project
cperm init      # pick modules: base + go + docker + git + strict-secrets
                # → writes .claude/compose.json and .claude/settings.json

# Add a capability
cperm add agent-teams   # adds module, recomposes automatically

# Detect drift from manual approvals
cperm status    # "⚠ 3 rules in settings.json not in compose"

# Import existing settings
cperm import .claude/settings.json
# "✓ 12 rules match 'base', ✓ 6 rules match 'go', ✗ 8 unmatched"
# → create new modules from unmatched rules
```

## Install

`cperm` isn't in a package manager yet, but it installs cleanly with Go.

**With `go install`** (no clone needed):

```bash
go install github.com/erikolson/cperm@latest
```

**From source:**

```bash
git clone https://github.com/erikolson/cperm
cd cperm
go build -o cperm .
```

Either way, the binary needs to be on your `PATH`. `go install` puts it in
`$(go env GOPATH)/bin`; for a source build, copy it there yourself:

```bash
mkdir -p "$(go env GOPATH)/bin" && cp cperm "$(go env GOPATH)/bin/"
```

If that directory isn't already on your `PATH`, add it (e.g.
`export PATH="$(go env GOPATH)/bin:$PATH"` in your shell profile). Requires Go 1.22+.
A Nix flake is included for a dev shell (`nix develop`).

> A Homebrew tap, prebuilt release binaries, and `nix run` are planned but not yet
> wired up — see [DESIGN_NOTES.md](DESIGN_NOTES.md).

## Quick Start

```bash
# 1. See what's available
cperm modules

# 2. Set up a project
cd your-project
cperm init

# 3. That's it. .claude/settings.json is composed and ready.
```

## How It Works

**Modules** are small JSON files containing permission rules for a specific toolchain or concern:

```json
{
  "name": "go",
  "description": "Go toolchain — build, test, lint",
  "requires": ["base"],
  "permissions": {
    "allow": ["Bash(go:*)", "Bash(golangci-lint:*)", "Bash(dlv:*)"]
  }
}
```

**compose.json** declares which modules a project uses:

```json
{
  "modules": ["base", "go", "docker", "git", "strict-secrets"],
  "override": {
    "allow": ["Bash(atlas:*)"]
  },
  "settings": {
    "defaultMode": "acceptEdits"
  }
}
```

`cperm compose` merges them into `.claude/settings.json` — deterministically, with deduplication and conflict detection.

## Built-in Modules

| Module | Description |
|--------|-------------|
| `base` | Core file ops, search, directory manipulation |
| `git` | Git + GitHub CLI (with `ask` on push) |
| `go` | Go toolchain |
| `node` | Node.js/TypeScript ecosystem |
| `python` | Python + pip + testing tools |
| `docker` | Docker operations (with `ask` on push) |
| `web` | WebFetch, WebSearch, curl, wget |
| `strict-secrets` | Deny access to .env, keys, credentials |
| `agent-teams` | Enable experimental agent teams + tmux |

## Commands

| Command | Description |
|---------|-------------|
| `cperm modules` | List available modules |
| `cperm modules show <n>` | Show module contents |
| `cperm init` | Interactive project setup |
| `cperm compose` | Rebuild settings.json from compose.json |
| `cperm add <module>` | Add a module to current project |
| `cperm remove <module>` | Remove a module from current project |
| `cperm status` | Detect drift between composed and actual settings |
| `cperm new <n>` | Create a new module interactively |
| `cperm edit <n>` | Open a module in $EDITOR |
| `cperm import [file]` | Decompose existing settings.json into modules |
| `cperm export` | Print composed output to stdout |
| `cperm compose --dry-run` | Preview without writing |

## The Workflow

The power of `cperm` is the **bottom-up discovery loop**:

1. **Use Claude Code normally** — approve permissions as they come up
2. **`cperm status`** — see what's drifted from your composed baseline
3. **`cperm import`** — decompose accumulated rules into reusable modules
4. **`cperm compose`** — reset to your declared state

Your module store becomes a curated reflection of how you actually work, not an abstract wishlist.

### Closing the loop automatically

That loop only helps if you remember to run it. [`examples/`](examples/) wires it into
Claude Code's hooks so it closes itself: a `Stop` hook runs `cperm status --json` and nudges
you when rules have drifted, and a `/cperm-promote` skill clusters the drifted approvals and
folds them into modules on your say-so. The deterministic part (detecting drift) stays in the
CLI; the judgment part (which module a rule belongs to, what's junk) lives in the skill.

## For Agent Teams

Agent teams inherit the lead's permissions at spawn time. Every permission prompt blocks a teammate's execution, so pre-approving generously matters more in multi-agent workflows. The `agent-teams` module enables the experimental feature and adds tmux permissions:

```bash
cperm add agent-teams
cperm compose
```

## Philosophy

- **Fragments are plain JSON** — no new language, no templating
- **The composition is the source of truth** — settings.json is a disposable output
- **Drift detection is first-class** — know when reality diverges from intent
- **Bottom-up discovery** — build modules from what you actually use, not what you imagine you'll need

## License

MIT
