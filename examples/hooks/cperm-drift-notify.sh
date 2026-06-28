#!/usr/bin/env bash
# cperm Stop hook — nudge when a project's settings.json has drifted from its
# composed cperm baseline.
#
# Non-intrusive by design: prints a single user-facing notice only when there
# are manual approvals to promote, and stays completely silent otherwise (no
# output, exit 0). It never blocks Claude or continues the conversation.
#
# Wire it up via the Stop hook in settings.json — see examples/settings.snippet.json.
set -euo pipefail

# Both tools are required; if either is missing, do nothing rather than error.
command -v jq    >/dev/null 2>&1 || exit 0
command -v cperm >/dev/null 2>&1 || exit 0

payload=$(cat)
cwd=$(printf '%s' "$payload" | jq -r '.cwd // empty')
[ -n "$cwd" ] || cwd=$PWD

# `cperm status --json` exits non-zero when the directory isn't a cperm project,
# which is the common case — skip quietly.
report=$(cd "$cwd" && cperm status --json 2>/dev/null) || exit 0

added=$(printf '%s' "$report" | jq -r '.addedCount // 0')
[ "$added" -gt 0 ] 2>/dev/null || exit 0

msg="cperm: ${added} rule(s) drifted from your composed baseline — run /cperm-promote to fold them into modules."
jq -n --arg m "$msg" '{systemMessage: $m, suppressOutput: true}'
