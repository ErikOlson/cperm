---
name: cperm-promote
description: Promote drifted Claude Code permission approvals back into reusable cperm modules. Use after cperm reports drift — i.e. manual approvals have accumulated in settings.json that aren't in any composed module.
disable-model-invocation: true
allowed-tools: Bash(cperm:*), Read, Edit, Write
---

# Promote drifted permissions into cperm modules

You are closing cperm's bottom-up loop: reconciling a project's accumulated,
manually-approved Claude Code permissions back into reusable modules. Apply
judgment — this is the part a plain diff can't do.

## Steps

1. **Read the drift.** Run `cperm status --json`. The `drift.added` arrays
   (`allow` / `ask` / `deny`) are rules present in `settings.json` but not in any
   composed module — the promotion candidates. If `addedCount` is 0, tell the
   user there's nothing to promote and stop.

2. **Cluster the added rules by intent**, the way a person would:
   - Group by tool/command family — e.g. `Bash(docker ...)` rules belong together,
     `Bash(go ...)` belong with the Go toolchain.
   - Run `cperm modules` to see what already exists (it also prints the module
     **store path**). Prefer *extending an existing module* over creating a new
     one when rules clearly belong to it.
   - **Drop one-off junk.** A frozen, hyper-specific command that will never recur
     — a long ad-hoc `grep`/`awk` pipeline, a one-time script path — is noise, not
     a reusable rule. Don't promote it; say you're skipping it and why.

3. **Propose before changing anything.** Show the user a concrete plan: which
   rules go into which module (existing or new), what you'd drop, and why. Wait
   for confirmation.

4. **Apply it.** cperm's `new`/`edit`/`import` commands are interactive, so don't
   drive those — edit the module store directly:
   - Modules are plain JSON at `<store>/<name>.json` (the store path is printed by
     `cperm modules`), shaped like
     `{"name": "...", "description": "...", "permissions": {"allow": [], "ask": [], "deny": []}}`.
   - **Extend** a module: Read its JSON, add the rules to the right array, Write it back.
   - **New** module: Write a new `<name>.json` to the store, then `cperm add <name>`
     to put it in this project's `.claude/compose.json`.
   - **Project-specific one-offs** that are real but don't merit a module: add them
     to the `override` block of `.claude/compose.json`.
   - Then run `cperm compose` followed by `cperm status` to confirm the drift is gone.

5. **Summarize** what moved where, and what you dropped.

Keep the module store a curated reflection of how the user actually works: broad,
reusable rules in modules; genuine one-offs in overrides; junk discarded.
