# Design Notes

Working notes for the cperm revival. This file holds things we *intend* to do but
haven't shipped yet, so the README and `--help` only claim what actually works today.

## Distribution (not yet shipped)

The goal is friction-free install once the repo is public. None of these exist yet —
they were advertised in an early README draft and have been pulled back to here until real:

- **Homebrew tap** — `brew install erikolson/tap/cperm`. Needs a `homebrew-tap` repo plus
  a goreleaser brew config to publish formulae on release.
- **Prebuilt release binaries** — GitHub Releases via goreleaser cross-compilation.
  `CLAUDE.md` references a `.goreleaser.yaml` that is not in the tree yet; it must be written.
- **`go install github.com/erikolson/cperm@latest`** — available now (repo is public and
  tagged); documented in the README. No longer deferred.
- **`nix run github:erikolson/cperm`** — needs the flake's `vendorHash` resolved (currently
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

   *Update (done): `cperm status` now reads overlay files via `render.OverlayPaths` and unions
   their permissions into the effective state, so approvals written to `settings.local.json`
   surface as drift, and `status --json` lists the `sources` it merged. Two follow-ups remain:
   the **sandbox block** in that file is still not reconciled (cperm only parses permissions),
   and drift matching is **exact-string** — a broad module rule like `Bash(git:*)` does not
   suppress a narrower accumulated approval like `Bash(git add *)`, so they show as drift until
   promoted or dropped. Pattern-aware (subsumption) matching would quiet that; for now the
   `/cperm-promote` skill is told to recognize rules already covered by a module.*

Note: confirm the exact precedence and that a `sandbox` block is honored in a checked-in
`.claude/settings.json` (vs. only `settings.local.json` / `~/.claude/settings.json`) before
building on this — the docs are explicit about the latter two scopes but do not show an
example of `sandbox` in project `settings.json`.

## Harness integration (future)

cperm is already shaped like a small reconciliation harness: declarative source of truth
(`compose.json` + modules), plan/apply (`compose` / `--dry-run` / `export`), drift detection
(`status`), and reconciliation (`import`). Today that loop is a manual CLI you have to remember
to run. The opportunity is to wire it into Claude Code's harness so the loop closes itself.

1. **Hook-driven drift→import loop.** A Claude Code `Stop` / `PostToolUse` hook watches the
   project's `settings.local.json`; when new approvals appear, it runs `cperm status` and nudges
   the user to promote drifted rules into a module ("3 rules drifted — base / git / new?"). This
   turns bottom-up discovery from "discipline you must remember" into something surfaced at the
   moment it's relevant — the thing that makes the compose/import diff actually converge in
   practice. Observed motivation: a single session accumulated eight near-duplicate
   `Bash(git …)` approvals that the existing `git` module's one `Bash(git:*)` rule would have
   subsumed entirely, plus one-off junk that `import` should drop rather than bucket. Related
   surfaces: package the commands as a Claude Code **skill / slash command** so the agent itself
   can invoke `import`/`compose`/`status`; a `SessionStart` hook that re-composes from
   `compose.json` for reproducible setup; a git pre-commit guard running `cperm status` to keep a
   team's policy from drifting.

2. **Compose the whole `settings.json`, not just `permissions`.** Extend cperm past the
   `permissions` block to also own `sandbox` (and other top-level keys), so one `cperm compose`
   provisions a project's permission rules *and* the sandbox `allowWrite` paths / `allowedDomains`
   a toolchain needs to function — e.g. a `go` module carrying both `Bash(go:*)` and the
   `proxy.golang.org` domain plus the build-cache write path. This is the real fix to the friction
   hit during the revival: the prompts we saw were sandbox *filesystem-boundary* prompts (build
   cache, `~/.config/cperm` store), not permission-rule prompts, so a permissions-only tool can't
   remove them. Couples with the `settings.local.json` precedence caveat above — if cperm writes a
   `sandbox` block to `settings.json` while the `/sandbox` panel writes `settings.local.json`, the
   local file wins and cperm's drift detection must account for it.

3. **cperm's interactive commands block agent-driven use.** `init`, `new`, `edit`, and
   `import` all prompt on stdin (or open `$EDITOR`), so an agent running the loop above
   can't drive them — the `cperm-promote` skill (see `examples/`) has to edit the
   module-store JSON directly and use only the non-interactive commands (`status`,
   `compose`, `add`, `remove`, `modules`, `export`). The fix is scriptable variants:
   `cperm new <name> --from-json -`, `cperm import --json`, a `--yes`/`--non-interactive`
   flag, etc., so the agent uses the CLI as its API rather than reaching around it. The
   `--json` flag on `status` was the first step in this direction.

## Two axes of decoupling

The Phase 3 work (internal `Policy` as source of truth, a `render.Renderer` adapter, and a
single `claudecode` implementation) is the seam that two larger directions build on. Both are
interface/implementation expansions of that same boundary — neither is in scope yet, but the
architecture is chosen so they don't require a redesign later.

- **Vertical — permissions → settings.** Today a module composes permission rules (allow/ask/
  deny) plus `env`, and arbitrary top-level keys ride along via the `settings` passthrough. The
  natural growth is to compose the *whole* settings document as a first-class concern (sandbox,
  hooks, MCP servers, `additionalDirectories`, …). The catch: this is a **merge-semantics**
  change, not a rename — flat array concat+dedup gives way to deep/shallow object merge with
  per-key strategies (last-wins / error / warn) for scalars like `defaultMode`. That is exactly
  the "general composition engine" in `CLAUDE.md`. When we commit to it, the internal `Policy`
  type broadens (likely renamed to `Config`/`Document`) behind the unchanged renderer seam. This
  is also why the `cperm` name becomes a misnomer; defer the rename until scope settles.

- **Horizontal — Claude → any agent.** The `Renderer` interface is agent-neutral by
  construction, so adding `render/codex`, `render/gemini`, etc. is the mechanism for targeting
  other agent brands. The easy part is the interface; the hard part is the **neutral model**.
  These agents express "what the agent may do" at very different granularities:
  - Claude Code — fine-grained rule strings (`Bash(git push:*)`, `Read(**/.env)`, MCP, sandbox), allow/ask/deny.
  - Codex CLI — coarse modes: an approval policy (untrusted / on-failure / never) plus a sandbox mode (read-only / workspace-write / full), in TOML.
  - Gemini CLI — tool allow/exclude lists (`coreTools` / `excludeTools`) plus MCP, in its own settings.json.

  A neutral core rich enough to project onto all of them is lossy in the coarse direction:
  Claude's per-command `ask` has no clean Codex equivalent, so a `codex` renderer must coarsen
  *and warn about what it dropped* rather than flatten silently. The payoff is the real DX goal —
  author one declarative policy, render it per-agent, feed whichever harness you're driving.

The throughline: do Phase 3's seam well now; both expansions become additive (a richer core
type behind the seam; more renderers behind the interface) rather than rewrites.

## See also

- `CLAUDE.md` — architecture, design principles, and the longer-term
  "general composition engine" direction.
