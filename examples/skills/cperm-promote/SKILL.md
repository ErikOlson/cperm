---
name: cperm-promote
description: Promote drifted Claude Code permission approvals back into reusable cperm modules. Use after cperm reports drift — i.e. manual approvals have accumulated in settings.json / settings.local.json that aren't in any composed module.
disable-model-invocation: true
allowed-tools: Bash(cperm:*), Read, Edit, Write
---

# Promote drifted permissions into cperm modules

You are closing cperm's bottom-up loop: reconciling a project's accumulated,
manually-approved Claude Code permissions back into reusable modules. Apply
judgment — this is the part a plain diff can't do. Drive cperm through its CLI;
all the commands below are non-interactive.

## Steps

1. **Read the drift.** Run `cperm status --json`. The `drift.added` arrays
   (`allow` / `ask` / `deny`) are rules present on disk but not in any composed
   module — the promotion candidates. (`sources` shows which files were merged,
   including `settings.local.json`.) If `addedCount` is 0, tell the user there's
   nothing to promote and stop.

2. **Cluster the added rules by intent**, the way a person would:
   - Group by tool/command family — e.g. `Bash(docker ...)` rules belong together.
   - `cperm modules` lists existing modules; `cperm import --json` reports which
     existing modules already cover which rules. Prefer *extending an existing
     module* over creating a new one when rules clearly belong to it.
   - Note that drift matching is exact-string: a broad rule already in a module
     (e.g. `Bash(git:*)`) will not have suppressed a narrower approval
     (`Bash(git add *)`), so such rules show as drift but are already covered —
     **drop them** rather than re-adding.
   - Also **drop one-off junk**: a frozen, hyper-specific command that will never
     recur (a long ad-hoc `grep`/`awk` pipeline, a one-time script path). Say
     what you're skipping and why.

3. **Propose before changing anything.** Show the user a concrete plan: which
   rules go into which module (existing or new), what you'd drop, and why. Wait
   for confirmation.

4. **Apply it via the CLI** (`cperm new` takes a non-interactive `--from-json`):
   - **Inspect** a module: `cperm modules show <name> --json` prints its JSON.
   - **Extend** a module: take that JSON, add the rules to the right permission
     array, and write it back with `cperm new <name> --from-json - --force`
     (pipe the updated JSON on stdin).
   - **Create** a module: pipe a fresh
     `{"description": "...", "permissions": {"allow": [...]}}` to
     `cperm new <name> --from-json -`, then `cperm add <name>` to add it to this
     project's `.claude/compose.json`.
   - **Project-specific one-offs** that are real but don't merit a module: add
     them to the `override` block of `.claude/compose.json` (edit the file).
   - Then run `cperm compose` followed by `cperm status` to confirm drift is gone.

5. **Summarize** what moved where, and what you dropped.

Keep the module store a curated reflection of how the user actually works: broad,
reusable rules in modules; genuine one-offs in overrides; junk and
already-covered rules discarded.
