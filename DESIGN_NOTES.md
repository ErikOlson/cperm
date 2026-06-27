# Design Notes

Working notes for the cperm revival. This file holds things we *intend* to do but
haven't shipped yet, so the README and `--help` only claim what actually works today.

## Distribution (not yet shipped)

The goal is friction-free install once the repo is public. None of these exist yet —
they were advertised in an early README draft and have been pulled back to here until real:

- **Homebrew tap** — `brew install erikmav/tap/cperm`. Needs a `homebrew-tap` repo plus
  a goreleaser brew config to publish formulae on release.
- **Prebuilt release binaries** — GitHub Releases via goreleaser cross-compilation.
  `CLAUDE.md` references a `.goreleaser.yaml` that is not in the tree yet; it must be written.
- **`go install github.com/erikmav/cperm@latest`** — works once the repo is public and tagged.
- **`nix run github:erikmav/cperm`** — needs the flake's `vendorHash` resolved (currently
  `null`, which will not build a real package; only the dev shell works today).

Until these land, the README documents build-from-source only.

## Sandbox integration (future)

Claude Code's OS-level **sandbox** is a second policy layer that sits next to the
permission rules cperm already composes. Permission rules (`allow`/`ask`/`deny`) decide
*whether* a tool runs; the sandbox decides *what a Bash command can touch* (filesystem
writes, network egress). They are orthogonal and configured by different keys. Two ideas
worth pursuing:

1. **Modules carry their required sandbox domains.** A toolchain often needs network egress
   to function inside the sandbox — e.g. `go mod tidy` must reach `proxy.golang.org` and
   `sum.golang.org`. The clean fix per Claude Code's docs is *not* the unsandboxed-fallback
   escape hatch but pre-allowing those domains in `sandbox.network.allowedDomains`, so the
   command succeeds in-sandbox with no prompt. A `go` module could declare both its
   `Bash(go:*)` permission rule *and* the domains it needs — so composing "go support" yields
   a toolchain that is both permitted and actually able to reach its registry, as one unit.
   This extends cperm from "compose the `permissions` block" to "compose the whole
   `settings.json`," which the `settings` passthrough already partially enables today
   (`compose.json` → `settings.sandbox.network.allowedDomains`).

2. **Drift detection must account for `settings.local.json` precedence.** The `/sandbox`
   panel writes to `.claude/settings.local.json`, which takes precedence over the
   `.claude/settings.json` cperm generates. cperm's `status`/drift logic currently reads only
   `settings.json`, so panel-driven sandbox config is invisible to it and the two files can
   silently diverge. Drift detection should at minimum be aware of the local-overrides file,
   even if cperm continues to own only `settings.json`.

Note: confirm the exact precedence and that a `sandbox` block is honored in a checked-in
`.claude/settings.json` (vs. only `settings.local.json` / `~/.claude/settings.json`) before
building on this — the docs are explicit about the latter two scopes but do not show an
example of `sandbox` in project `settings.json`.

## See also

- `CLAUDE.md` — architecture, design principles, and the longer-term
  "general composition engine" direction.
