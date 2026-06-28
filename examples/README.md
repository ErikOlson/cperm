# Examples — the bottom-up reconciliation loop

These artifacts wire cperm into Claude Code's harness so the
"use → drift → promote → recompose" loop closes itself instead of relying on you
to remember to run it.

```
you work in Claude Code
   │  (approve permissions as they come up — they pile up in settings.json)
   ▼
Stop hook runs `cperm status --json`
   │  drift detected? → one-line nudge: "N rules drifted — run /cperm-promote"
   ▼
you run /cperm-promote
   │  the skill clusters the drifted rules, proposes which module each belongs
   │  to (dropping junk), and on your OK edits the module store + recomposes
   ▼
modules now reflect how you actually work; drift returns to zero
```

The two halves are deliberately split by responsibility: the **hook** is a dumb,
deterministic notifier (no judgment, never blocks); the **skill** is where the
judgment lives (clustering, naming, dropping one-offs) and only runs when you
ask for it.

## Contents

| File | What it is |
| ---- | ---------- |
| `hooks/cperm-drift-notify.sh` | `Stop` hook: nudges (via `systemMessage`) when `addedCount > 0`, silent otherwise. Needs `jq` and `cperm` on `PATH`. |
| `settings.snippet.json` | The `hooks.Stop` block to merge into your `settings.json`. |
| `skills/cperm-promote/SKILL.md` | The `/cperm-promote` skill. `disable-model-invocation: true` — only you can trigger it. |

## Install

Per-project (this project only):

```sh
# 1. Hook script
mkdir -p .claude/hooks
cp examples/hooks/cperm-drift-notify.sh .claude/hooks/
chmod +x .claude/hooks/cperm-drift-notify.sh

# 2. Skill
mkdir -p .claude/skills/cperm-promote
cp examples/skills/cperm-promote/SKILL.md .claude/skills/cperm-promote/

# 3. Merge examples/settings.snippet.json into .claude/settings.json
#    (the hooks.Stop block). $CLAUDE_PROJECT_DIR expands to the project root;
#    adjust the command path if you put the script elsewhere.
```

For every project, install the skill under `~/.claude/skills/cperm-promote/` and
the hook under `~/.claude/settings.json` instead.

## Requirements

- `cperm` on your `PATH`
- `jq` (the hook uses it to read the payload and cperm's JSON)
- a project with a `.claude/compose.json` (run `cperm init`)

The hook does nothing — no output, exit 0 — in any directory that isn't a cperm
project, so it's safe to enable globally.
