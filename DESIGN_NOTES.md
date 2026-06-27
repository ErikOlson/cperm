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

## See also

- `CLAUDE.md` — architecture, design principles, and the longer-term
  "general composition engine" direction.
