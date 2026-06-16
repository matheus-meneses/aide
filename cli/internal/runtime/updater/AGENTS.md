# updater

## Purpose

Version checking and in-place self-update for both the CLI and the macOS desktop
app. Detects how aide was installed at runtime and routes the update to a safe
strategy, so the same `aide update` / "Update now" action works on every channel.

## Exported API

- `CheckOnce(currentVersion string)` — throttled (12h) check that prints the upgrade banner.
- `AutoCheck(version string, autoApply bool)` — throttled check; when `autoApply` is true and the install is a CLI method (script/Homebrew formula), applies the update and prints progress to stderr; otherwise prints the banner.
- `LatestRelease() (Release, error)` — latest non-prerelease tag + notes (markdown) + html URL.
- `LatestUpgrade(current string) (Release, error)` — channel-aware: stable builds get the latest stable; prerelease builds also consider newer prereleases (so an rc moves to a newer rc).
- `ReleaseByTag(tag string) (Release, error)` — a specific release (incl. prereleases) by tag.
- `IsNewer(latest, current string) bool` — semver comparison, prerelease-precedence aware (rc.9 > rc.8; final > rc.N).
- `DetectMethod(version string) Method` — classifies the install (`dev`, `script`, `homebrew-formula`, `homebrew-cask`, `manual-app`, `unknown`).
- `Method.CanSelfUpdate() bool` — whether `Apply` can update this install in place.
- `Apply(ctx, currentVersion, method, prog) (Result, error)` — performs the update; `Result.RestartNow` is true for the app flow (caller must quit so the detached helper can swap the bundle and relaunch).
- `Progress func(line string)` — progress sink used by `Apply`/`AutoCheck`.
- `DownloadFile` / `DownloadToPath` — file download utilities.

## Update strategy per method

- `script` — download `aide_<os>_<arch>`, verify its `.sha256`, `chmod 0755`, atomic `os.Rename` over the running binary.
- `homebrew-formula` — `brew update` + `brew tap matheus-meneses/aide` + `brew upgrade aide`, streaming output. brew owns the file; we never overwrite it and never need admin.
- `homebrew-cask` — a detached helper runs `brew upgrade --cask aide` after the app quits, then relaunches (brew can't replace a running app).
- `manual-app` — download + verify the `Aide-<ver>.dmg`, stage the bundle, and a detached helper swaps `/Applications/Aide.app`, strips quarantine, ad-hoc re-signs, and relaunches once the app exits. Falls back to `osascript ... with administrator privileges` when `/Applications` is not writable.

## Important invariants

- Skips entirely when `version == "dev"`.
- `shouldCheck`/`markChecked` throttle to once per 12h via `~/.aide/.last_version_check`.
- Downloads are sha256-verified against the published `<asset>.sha256` before being applied.
- Release base/download URLs honor `AIDE_RELEASE_URL`; repo slug honors `AIDE_REPO`.
- App self-update (`homebrew-cask`, `manual-app`) is darwin-only (`apply_darwin.go`; `apply_other.go` returns an error elsewhere).
- The agent's `RequestRestart()` (set by the desktop app via `SetRestartHandler`) is what quits the app so a staged update can finish.

## Pitfalls

- Network/update failures in `AutoCheck`/`CheckOnce` are swallowed by design (non-blocking).
- `/releases/latest` excludes prereleases; use `LatestUpgrade` so prerelease builds still see newer rc tags (it lists `/releases` and picks the newest by semver).
- The released CLI must be built with `-X aide/cli/internal/agent.Version=...` (see `release.yml`) or `aide ui` reports `dev` and never offers updates.

## Relations

- Used by: `cmd/aide` (banner + `aide update`), `agent/api` (`/api/version`, `/api/update`), `cmd/aide-app` (restart hook).
- Leaf package (no internal aide deps).
