# cperm — Composable Claude Code Permissions

## What This Is

A CLI tool that brings Nix-inspired declarative composition to Claude Code permission management. Instead of manually curating `.claude/settings.json` files per project, users define reusable **permission modules** and compose them per-project with drift detection.

## Architecture

```
cperm/
  main.go                          # Entry point
  cmd/                             # CLI commands (cobra)
    root.go                        # Root command, shared helpers, lipgloss styles
    modules.go                     # List/show available modules
    init.go                        # Interactive project setup
    compose.go                     # Build settings.json from compose.json
    addremove.go                   # Add/remove modules from compose.json
    status.go                      # Drift detection
    import.go                      # Decompose existing settings.json into modules
    newmodule.go                   # Create new modules interactively
    export.go                      # Print composed output to stdout
  internal/
    model/model.go                 # Core types: Module, ComposeFile, ClaudeSettings, ComposedResult
    store/store.go                 # Module store CRUD (~/.config/cperm/modules/)
    store/embed.go                 # go:embed for built-in modules
    store/builtins/                # Embedded starter modules (JSON)
    composer/composer.go           # Merge engine: dependency resolution, dedup, conflict detection
    importer/importer.go           # Reverse-engineer modules from existing settings.json
  modules/                         # Source copies of built-in modules (also in store/builtins/)
  flake.nix                        # Nix flake for building and distribution
  .goreleaser.yaml                 # Cross-compilation and release automation
```

### Key Internal Boundaries

- **`internal/engine/`** (future) — The general-purpose composition engine will be extracted from `composer/` and `store/`. Design all composition logic to be format-agnostic where possible.
- **`internal/model/`** — Domain types. `Module` and `ComposeFile` are the data model. `ComposedResult` carries merge metadata (dedup counts, conflicts).
- **`internal/store/`** — Filesystem operations for the module store. Built-in modules are embedded via `go:embed` in `embed.go` and seeded on first use (skipping existing user modules).
- **`internal/composer/`** — The merge engine. Resolves module dependencies (topological sort), concatenates permission arrays, deduplicates, detects conflicts (same rule in allow + deny).
- **`internal/importer/`** — The adoption on-ramp. Matches existing settings.json rules against known modules and identifies unmatched rules for promotion into new modules.

## Data Flow

```
User's module store          Project compose.json
(~/.config/cperm/modules/)   (.claude/compose.json)
        │                            │
        └──────────┬─────────────────┘
                   │
            composer.Compose()
                   │
           ┌───────┴────────┐
           │  Resolve deps  │  (topological sort of module.Requires)
           │  Merge arrays  │  (concat allow/deny/ask from each module in order)
           │  Apply override│  (compose.json override block applied last)
           │  Dedup         │  (uniqueStrings preserving order)
           │  Detect conflicts│ (same rule in multiple arrays)
           └───────┬────────┘
                   │
            .claude/settings.json    (Claude Code reads this natively)
```

## Design Principles

1. **Fragments are plain JSON, not code.** No DSL, no templating language.
2. **Merge strategy is simple and predictable.** Arrays concatenate and deduplicate. Deny always wins at the Claude Code level.
3. **Plan before apply.** `--dry-run` and `export` show what would change.
4. **Drift detection is first-class.** `status` compares composed state vs actual file.
5. **Bottom-up discovery is the killer feature.** `import` decomposes existing configs into reusable modules. The workflow is: use Claude Code → accumulate permissions → import → promote to modules → compose.
6. **The composition is the source of truth.** `settings.json` is a disposable output.
7. **Built-in modules ship embedded** and seed on first use but never overwrite user customizations.

## Key Design Decisions

- **`compose.json` is JSON, not a plain text list.** Slightly more friction but supports the `override` block and `settings` passthrough without a second format.
- **Modules can declare `requires` (dependencies).** Resolved via depth-first topological sort with cycle detection. Dependencies are processed before the module that requires them.
- **The `override` block in compose.json** exists for project-specific one-offs that don't merit their own module. Applied last in the merge order.
- **`settings` in compose.json** is a passthrough map — keys are written to the top level of the output settings.json (e.g., `defaultMode`, custom keys).
- **Store location is `~/.config/cperm/modules/`** — deliberately separate from `~/.claude/` to avoid colliding with Claude Code's own config namespace.

## Built-in Modules

base, git, go, node, python, docker, web, strict-secrets, agent-teams

These live in `internal/store/builtins/` and are embedded into the binary. On first `getStore()` call, any missing builtins are installed to the user's store.

## Dependencies

- **cobra** — CLI framework
- **lipgloss** — Terminal styling
- **bubbletea** — Planned for interactive TUI (v0.2 — currently using simple stdin prompts)

## What's Shipped (v0.1)

All commands: modules, modules show, init, compose, add, remove, status, new, edit, import, export

## What's Deferred (v0.2)

- `diff` command — detailed preview of what compose would change
- `doctor` command — store integrity checks, orphaned module detection
- `validate` command — check compose.json and modules for errors
- `search` command — search modules by name/description/permission pattern
- Bubbletea interactive multi-select in `init` (replace current number-input UX)
- Module versioning and lock file
- Community module registry

## Future Direction: General Composition Engine

cperm is the domain-specific "wedge" for a more general tool. The architecture is designed for extraction:

- `internal/composer/` and `internal/store/` contain the general composition logic
- The general tool would support multiple **stores** (one per config domain) with per-store **merge strategies**
- Merge strategy vocabulary: deep/shallow object merge, concat/concat-unique/replace/by-key array merge, last-wins/first-wins/warn/error conflict resolution
- Format support: JSON, YAML, TOML, INI, text (line-based)
- Prior art studied: Nix (overlays, hermeticity), Terraform (plan/apply, drift detection), Chezmoi (template-based dotfiles), Kustomize (base+overlay, strategic merge patch), Jsonnet/CUE (typed constraints), Ansible (role dependencies)

## Commands and Build

```bash
go mod tidy                    # Fetch dependencies
go build -o cperm .            # Build
go test ./...                  # Run tests (when written)
goreleaser release --snapshot  # Test release build
```

## Style Notes

- Go standard project layout
- Errors wrap with context using `fmt.Errorf("context: %w", err)`
- User-facing output uses lipgloss styles defined in cmd/root.go
- Interactive prompts use bufio.Reader (v0.1), migrating to bubbletea (v0.2)
- All file writes include trailing newline
- JSON output uses 2-space indent
