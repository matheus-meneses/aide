# Security Policy

## Supported Versions

Security fixes are applied to the latest released version. Always upgrade to the
most recent release before reporting an issue:

| Version        | Supported          |
| -------------- | ------------------ |
| Latest release | :white_check_mark: |
| Older releases | :x:                |

## Reporting a Vulnerability

Please report security vulnerabilities **privately** so we can investigate and
ship a fix before details are public.

- Preferred: open a private advisory via GitHub's
  [Report a vulnerability](https://github.com/matheus-meneses/aide/security/advisories/new)
  flow. This keeps the report confidential and lets us collaborate on a fix.
- Do **not** open a public issue, pull request, or discussion for a suspected
  vulnerability.

When reporting, include as much of the following as you can:

- A description of the vulnerability and its impact.
- Steps to reproduce, or a proof-of-concept.
- Affected version(s), platform, and configuration.
- Any suggested remediation.

## Response Expectations

- We aim to acknowledge new reports within 5 business days.
- We will keep you informed of progress while we validate and fix the issue.
- Once a fix is released, we will credit reporters who wish to be acknowledged.

## Scope

This policy covers the `aide` CLI, the desktop app, and the Go/Python SDKs in
this repository. Plugins run untrusted third-party code inside an OS sandbox
(see the Security section of the README); issues that require a plugin to
already have been granted capabilities it declared are expected behavior rather
than vulnerabilities in aide itself.
