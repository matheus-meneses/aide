# Releasing

This repo ships three things from a single `v*` tag:

- **CLI binaries** for macOS/Linux/Windows (`aide_<os>_<arch>`)
- **macOS desktop app** as `Aide-<version>.dmg`
- **Homebrew tap** (`matheus-meneses/homebrew-aide`) formula + cask, bumped automatically

The Python SDK releases independently from `sdk-v*` tags.

## Cut a release

1. Update `VERSION` to the new version (e.g. `1.4.0`, no leading `v`).
2. Commit and merge to `main`.
3. Tag and push:

```sh
git tag v1.4.0
git push origin v1.4.0
```

The `Release` workflow runs:

- `verify-version` — fails fast unless `VERSION` matches the tag.
- `release` — cross-builds the CLI, checksums, build-provenance + SBOM attestations, uploads assets + `install.sh`.
- `release-macos-app` — builds the universal `Aide.app`, packages the DMG, checksums, attestations, uploads.
- `update-homebrew-tap` — renders the formula + cask with the real checksums and pushes them to the tap.

## Required secrets

| Secret | Used for | Required? |
| --- | --- | --- |
| `HOMEBREW_TAP_TOKEN` | Push to `matheus-meneses/homebrew-aide` (PAT, `contents:write` on the tap) | For tap auto-bump |
| `MACOS_CERTIFICATE` | base64 of the Developer ID `.p12` | Only for signed builds |
| `MACOS_CERTIFICATE_PWD` | `.p12` password | Only for signed builds |
| `MACOS_KEYCHAIN_PASSWORD` | temp keychain password | Only for signed builds |
| `MACOS_DEVELOPER_ID` | `Developer ID Application: Name (TEAMID)` | Only for signed builds |
| `MACOS_NOTARIZE_PROFILE` | `notarytool` keychain profile name | Only for notarized builds |

Without the macOS signing secrets the app is **ad-hoc signed** and the cask strips
the quarantine attribute on install, so it still launches.

## One-time setup

- Create an empty public repo `matheus-meneses/homebrew-aide` and add the
  `HOMEBREW_TAP_TOKEN` secret. The first tagged release populates it.
- Tap source of truth lives in `packaging/homebrew/` (`render.sh` + reference
  `Formula/` and `Casks/`).

## Signing later (optional)

When a Developer ID is available, add the macOS secrets above. The
`release-macos-app` job imports the certificate into a temp keychain and the
packaging script (`cli/cmd/aide-app/packaging/build-macos.sh`) signs and, if
`MACOS_NOTARIZE_PROFILE` is set, notarizes and staples automatically.

## Verifying attestations

```sh
gh attestation verify aide_darwin_arm64 --repo matheus-meneses/aide
gh attestation verify Aide-1.4.0.dmg --repo matheus-meneses/aide
```
