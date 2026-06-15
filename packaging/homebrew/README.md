# Homebrew tap source

This directory is the source of truth for the [`matheus-meneses/homebrew-aide`](https://github.com/matheus-meneses/homebrew-aide)
tap. The files here are rendered and pushed to that tap automatically by the
`update-homebrew-tap` job in `.github/workflows/release.yml` on every `v*` tag.

## Layout

- `render.sh` — renders `Formula/aide.rb` (CLI) and `Casks/aide.rb` (desktop app)
  for a given version and set of release checksums.
- `Formula/aide.rb` — reference copy of the CLI formula (placeholder version).
- `Casks/aide.rb` — reference copy of the desktop-app cask (placeholder version).

## Installing (once the tap exists)

```sh
brew tap matheus-meneses/aide
brew install aide          # CLI
brew install --cask aide   # desktop app (unsigned; quarantine stripped on install)
```

## One-time tap setup

Create an empty public repo named `homebrew-aide` under `matheus-meneses`. The
release pipeline pushes the rendered `Formula/` and `Casks/` into it using the
`HOMEBREW_TAP_TOKEN` secret (a PAT with `contents:write` on that repo).

## Rendering locally

```sh
# sums-dir holds <asset>.sha256 files for each released artifact
./render.sh 1.4.0 ./sums ./out
```
