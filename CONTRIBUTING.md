# Contributing to aide

## Dev setup

1. Clone the repo and build the sandbox binary:

```bash
git clone https://github.com/matheus-meneses/aide.git
cd aide
make dev
```

`make dev` compiles the binary, copies it to `~/.aide-sandbox/bin/aide`, and creates a wrapper script
`~/.aide-sandbox/bin/aide-dev` that pre-sets `AIDE_HOME=~/.aide-sandbox` and `AIDE_SDK_PATH=<repo>/sdk/python`.

2. Install a plugin in the sandbox and run it:

```bash
aide-dev plugin install --local aide-plugins/plugins/jira
aide-dev config source add
aide-dev run
```

3. Run the full verify gate before committing:

```bash
make verify
```

This must pass. It runs all linters, type-checkers, and tests across Go, Python, and the TypeScript frontend.

## The verify gate

| Target         | What it runs                                            |
|----------------|---------------------------------------------------------|
| `make fmt`     | gofumpt (Go), ruff format (Python), prettier (frontend) |
| `make go-lint` | golangci-lint in `cli/`                                 |
| `make go-test` | `go test -race ./...` in `cli/`                         |
| `make go-vuln` | govulncheck in `cli/`                                   |
| `make py-lint` | ruff check in `sdk/python` and `aide-plugins/plugins`   |
| `make py-type` | mypy on `aide_sdk`                                      |
| `make py-test` | pytest in `sdk/python`                                  |
| `make fe-lint` | tsc typecheck + eslint in the frontend                  |
| `make verify`  | all of the above in sequence                            |

`make verify` must pass before pushing any branch.

## Commit convention

```
<type>(<scope>): <description>
```

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `build`

**Scopes:** `cli`, `runtime`, `sandbox`, `store`, `render`, `agent`, `runner`, `keychain`, `prompt`, `updater`, `sdk`,
`plugin`, `frontend`

Examples:

- `feat(plugin): add browser capability flag to sandbox`
- `fix(runner): correctly handle empty scrape_team response`
- `docs(sdk): update BaseScraper contract in AGENTS.md`

## Code conventions

- **No narrating comments.** Comments explain *why*, never *what*.
- **Wrap errors with context.** `fmt.Errorf("loading config: %w", err)` — never bare `return err` without context.
- **Go:** errors compared with `errors.Is`/`errors.As`, never `==`. Context propagated through call chains.
- **Python:** all logging to stderr. Never write to stdout — it is reserved for the plugin JSON protocol.
- **TypeScript:** strict mode, no `any` without a documented reason.
- **No `//nolint`, `# noqa`, or `eslint-disable` as the first fix.** Fix the root cause.

## PRs

- Keep PRs focused on a single scope.
- `make verify` must pass.
- Link the issue or describe the motivation in the PR description.
