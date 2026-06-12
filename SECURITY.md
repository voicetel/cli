# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 0.2.x   | ✅        |
| < 0.2   | ❌        |

## Reporting a Vulnerability

Please **do not** open a public issue for security vulnerabilities.

Use GitHub's private vulnerability reporting for this repository:
**Security → Report a vulnerability** (or
<https://github.com/voicetel/cli/security/advisories/new>).

Include, where possible:

- A description of the issue and its impact
- Steps to reproduce or a proof of concept
- Affected version(s) and configuration

You can expect an acknowledgement within a few business days. Please
allow reasonable time for a fix before any public disclosure.

## Scope Notes

The CLI persists per-profile credentials (AccountSid + API key) to
the user's config directory and constructs authenticated HTTP
requests against the VoiceTel REST API via the VoiceTel Go SDK.
Hardening expectations:

- Keep the config file readable only by the owning user (default
  permissions are 0600).
- Do not paste the API key into shared shell history; use the
  `-x` flag with a pipe or environment-loaded value where possible.
- The REPL never echoes the API key back to the terminal once set,
  but it is held in process memory for the duration of the session.
- The Go SDK underlying the CLI pins TLS via `crypto/tls` defaults
  (TLS 1.2+, session ticket cache enabled).

Out of scope: vulnerabilities in a forked / modified build of the
CLI binary, or credential leakage caused by a debugger / process
inspector with privileged access to the running process.
