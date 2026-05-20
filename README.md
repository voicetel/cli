# 📟 VoiceTel CLI

The official interactive REPL for the [VoiceTel REST API](https://voicetel.com/docs/api/v2.2/) — provision numbers, place orders, validate e911, send messages, and manage your account, all from a single keystroke-friendly prompt on top of the [VoiceTel Go SDK](https://github.com/voicetel/go-sdk).

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![Go](https://img.shields.io/badge/go-1.22%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macos%20%7C%20windows-lightgrey)

![REPL demo](docs/demo.gif)

## 📚 Table of Contents

- [Features](#-features)
- [Installation](#-installation)
- [Quickstart](#-quickstart)
- [One-shot mode (`-x`)](#-one-shot-mode--x)
- [Authentication](#-authentication)
- [Configuration](#-configuration)
- [Environment variables](#-environment-variables)
- [Building from source](#-building-from-source)
- [Command Reference](#-command-reference)
- [Examples](#-examples)
- [API Documentation](#-api-documentation)
- [Contributors](#-contributors)
- [Sponsors](#-sponsors)
- [License](#-license)

## ✨ Features

### 🧑‍💻 REPL-First UX
- **Interactive prompt** — drops you straight into a `voicetel>` shell. No subcommands to memorise, no flag-soup one-liners.
- **Line editing** powered by [`github.com/chzyer/readline`](https://github.com/chzyer/readline) — Emacs-style movement, reverse-i-search, the works.
- **Persistent history** at `~/.voicetel/history` so you can `↑` back into yesterday's session.
- **Tab autocomplete** across the entire command tree — top-level resources, sub-commands, and known argument shapes.
- **Pretty-printed JSON** by default. Errors print in red when you're on a TTY; plain otherwise (so logs stay clean).

### 🛡️ Built on a Typed SDK
- Sits on top of the official [`voicetel/go-sdk`](https://github.com/voicetel/go-sdk) — every command is a one-line wrapper around a typed SDK method, so the CLI inherits the SDK's retry/backoff, structured errors, and rate-limit handling.
- **Persistent config** at `~/.voicetel/config.toml` — log in once, stay logged in. Passwords are **never** persisted; only the API key is.
- Sends `voicetel-cli/<version>` as the user-agent so requests are attributable in your logs.

### 📞 Complete API Coverage
- **Numbers** — list, get, add, remove, route, translate, CNAM, LIDB, fax, forward, SMS, messaging campaigns, port-out PIN, account moves.
- **Account** — profile, sub-accounts, CDRs, credits, payments, MRC, registration, password recovery.
- **e911** — record provisioning, address validation, lookup, removal.
- **Gateways** — list, create, update, delete, view bound numbers.
- **Messaging** — SMS & MMS sending, message history, 10DLC brand and campaign registration, per-number messaging state.
- **Lookups** — CNAM and LRN dips.
- **iNumbering** — inventory search, coverage queries, number orders, port-in submissions, port-out availability.
- **Support** — ticket create / read / update / delete, threaded messages, replies.
- **ACL** — IP allowlist management.
- **Authentication** — switch between Digest, IP-only, or hybrid modes; rotate passwords.

### 📦 Single Static Binary
- One binary, no runtime dependencies. Distributed via `go install` and pre-built releases.

## 🚀 Installation

### Via `go install`

```bash
go install github.com/voicetel/cli@latest
```

Requires Go 1.22 or later. The binary lands at `$GOBIN/voicetel-cli` (or `$GOPATH/bin/voicetel-cli`).

### Pre-built binaries

Grab a release from [GitHub Releases](https://github.com/voicetel/cli/releases) — macOS (amd64/arm64), Linux (amd64/arm64/386/arm), Windows (amd64/arm64), and FreeBSD (amd64) are all built per release.

```bash
# Linux amd64 example — adjust the archive name for your platform.
curl -sL https://github.com/voicetel/cli/releases/latest/download/voicetel-cli_0.2.0_linux-amd64.tar.gz | tar xz
sudo mv voicetel-cli_0.2.0_linux-amd64/voicetel-cli /usr/local/bin/
```

### From source

```bash
git clone https://github.com/voicetel/cli && cd cli
make build           # → bin/voicetel-cli (local platform)
make install         # → $GOPATH/bin/voicetel-cli
```

See [Building from source](#-building-from-source) for the full make-target list (including `make release` for the per-platform archives).

## 🏁 Quickstart

```text
$ voicetel-cli
VoiceTel CLI 0.2.0  —  type `help` for commands, `exit` to quit.
Endpoint: https://api.voicetel.com
No API key configured. Run `login <username> <password>` or `set api-key <key>`.

voicetel> login 1000000001 hunter2
Logged in. API key installed and saved.

voicetel> account get
{
  "username": "1000000001",
  "name": "Acme Co",
  "email": "ops@acme.example",
  "cash": 142.18,
  "callerId": "2015551234",
  "timezone": "America/Chicago"
}

voicetel> numbers list
{
  "numbers": [
    {"number": "2015551234", "route": 4, "cnam": true, "smsEnabled": true},
    {"number": "2015555678", "route": 4, "cnam": false, "smsEnabled": false}
  ]
}

voicetel> exit
```

## ⚡ One-shot mode (`-x`)

Run a single command non-interactively and exit. Useful for shell pipelines, cron jobs, and ad-hoc scripting — no REPL, no banner, no history file touched.

```bash
# Authenticate via env vars, then list numbers, then exit.
export VOICETEL_USERNAME=1000000001
export VOICETEL_PASSWORD=hunter2
voicetel-cli -x 'account numbers'

# Or use a pre-fetched API key directly:
export VOICETEL_API_KEY=abcdef0123456789abcdef0123456789
voicetel-cli -x 'numbers list'

# Pipe the JSON output into jq:
voicetel-cli -x 'account get' | jq .cash
```

The command string after `-x` is parsed the same way as a REPL line — group + subcommand + arguments (e.g. `acl add 203.0.113.0/24`, `messaging campaigns list`, `lookups cnam 2155551234`). Exit code is 0 on success, 1 on any error (including auth failures and rate-limit hits).

In one-shot mode, **`Ctrl-C` cancels the in-flight request** (the REPL leaves Ctrl-C to readline; one-shot mode handles it directly).

## 🔑 Authentication

Two ways to install an API key:

1. **`login <username> <password>`** — exchanges credentials for a fresh 32-hex bearer token via `POST /v2.2/account/api-key`. The token is persisted to `~/.voicetel/config.toml`; the password never is.

   This endpoint shares the **6 req/hr/IP** rate limit with the rest of `/account/*`.

2. **`set api-key <key>`** — installs an existing key directly. Useful if you've fetched one out of band.

```text
voicetel> login 1000000001 hunter2
Logged in. API key installed and saved.

voicetel> whoami
{
  "apiKey": "abcd...4321",
  "baseURL": "https://api.voicetel.com",
  "rateLimits": "account/cdr, account/mrc, account/payments, account/registration, account/api-key (login): 6 req/hr/IP"
}
```

Don't have credentials yet? Get them at **[voicetel.com/docs/api/v2.2/credentials](https://voicetel.com/docs/api/v2.2/credentials/)**.

## ⚙️ Configuration

The CLI reads its persistent config from `~/.voicetel/config.toml`. A typical file looks like:

```toml
api_key  = "abcdef0123456789abcdef0123456789"
base_url = "https://api.voicetel.com"
```

- `api_key` — the 32-hex bearer token. Written on `login` and `set api-key`.
- `base_url` — override the API endpoint (mostly for staging/sandbox builds). Defaults to `https://api.voicetel.com`.

The file is written atomically (temp sibling + `fsync` + `rename`) and chmodded to `0600`.

You can override either value on launch:

```bash
voicetel-cli --api-key=abcdef0123456789abcdef0123456789
voicetel-cli --base-url=https://staging.voicetel.com
```

Command-line flags do **not** persist; they win for the duration of the session only.

History lives at `~/.voicetel/history` (also chmodded to `0600` by readline).

## 🌳 Environment variables

The CLI honors four environment variables. Precedence is **flag > env > config**.

| Variable | Purpose |
|---|---|
| `VOICETEL_API_KEY` | 32-hex bearer token — installed directly on the client; no `login` round-trip. |
| `VOICETEL_USERNAME` | Numeric account id. Paired with `VOICETEL_PASSWORD`, triggers a `login` at startup. |
| `VOICETEL_PASSWORD` | Password — paired with `VOICETEL_USERNAME`; never persisted to `config.toml`. |
| `VOICETEL_BASE_URL` | Override the API endpoint (rare — staging / sandbox builds). |

```bash
# Headless CI script: use env vars, run a single command, exit.
export VOICETEL_USERNAME=1000000001
export VOICETEL_PASSWORD=$(vault read -field=password secret/voicetel)
voicetel-cli -x 'account info'
```

When `VOICETEL_USERNAME` and `VOICETEL_PASSWORD` are both set and no API key is available, the CLI calls the `login` endpoint at startup. The exchanged key is installed on the client but **not** written to `~/.voicetel/config.toml` (env-driven runs are intentionally ephemeral — they should not mutate user state).

## 🔨 Building from source

The repo ships a `Makefile` that produces pure-Go binaries (`CGO_ENABLED=0`) for 9 platforms — no per-target C toolchains needed.

```bash
make build       # local platform → ./bin/voicetel-cli
make build-all   # all 9 platforms → ./dist/voicetel-cli_<version>_<os>-<arch>/voicetel-cli[.exe]
make release     # build-all + per-platform .tar.gz / .zip archives in ./dist/
make test        # go test ./...
make vet         # go vet ./...
make install     # CGO_ENABLED=0 go install . → $GOPATH/bin/voicetel-cli
make clean       # rm -rf ./bin ./dist
```

Supported cross-compile targets:

- **macOS**: `darwin/amd64`, `darwin/arm64`
- **Linux**: `linux/amd64`, `linux/arm64`, `linux/386`, `linux/arm`
- **Windows**: `windows/amd64`, `windows/arm64`
- **FreeBSD**: `freebsd/amd64`

To add or remove a target, edit the `PLATFORMS` variable at the top of the `Makefile`.

## 🗺️ Command Reference

```text
help [topic]              Show help for the REPL or a specific command/topic.
exit | quit               Leave the REPL (Ctrl-D also works).
login <username> <password>     Exchange creds for an API key (rate-limited 6/hr/IP).
set api-key <key>         Install an existing 32-hex bearer.
set base-url <url>        Override the API endpoint (rare; usually production).
whoami                    Show the current API key, base URL, and rate-limit caveats.
clear                     Clear the screen.

account get
account update <json>     Body is parsed as JSON for the AccountPutRequest.
account add <json>
account signup <json>
account cdr [start] [end]
account credits
account recurring-charges  (alias: account mrc)
account payments
account registration
account recover <json>

acl list
acl add <json>
acl remove <json>

authentication get
authentication update <json>

e911 list
e911 create <json>
e911 validate <json>
e911 get <dn>
e911 provision <dn> <json>
e911 remove <dn>

gateways list
gateways add <json>
gateways get <id>
gateways update <id> <json>
gateways remove <id>
gateways numbers <id>

inumbering search-inventory [--npa --nxx --state --rate-center --contains --ends-with --limit]
inumbering coverage [--state --rate-center]
inumbering order <json>
inumbering ports
inumbering port <id>
inumbering submit-port <json>
inumbering port-availability <number>

lookups cnam <number>
lookups lrn <number> <ani>

messaging history [--number --start --end --type]
messaging send <json>
messaging create-brand <json>
messaging campaign-status
messaging create-campaign <json>
messaging numbers-state [--numbers=2015551234,2015551235]

numbers list
numbers add <json>
numbers get <number>
numbers remove <number>
numbers move <number> <json>
numbers release <number>
numbers set-route <number> <json>
numbers set-translation <number> <json>
numbers set-cnam <number> <json>
numbers set-lidb <number> <json>
numbers get-fax <number>
numbers set-fax <number> <json>
numbers remove-fax <number>
numbers set-forward <number> <json>
numbers remove-forward <number>
numbers get-sms <number>
numbers set-sms <number> <json>
numbers remove-sms <number>
numbers get-messaging <number>
numbers patch-messaging <number> <json>
numbers assign-campaign <number> <json>
numbers unassign-campaign <number>
numbers bulk-unassign-campaign <2015551234,2015551235>
numbers set-port-out-pin <number> <json>

support list
support create <json>
support get <id>
support update <id> <json>
support delete <id>
support messages <id>
support reply <id> <json>
```

`<json>` is the rest of the line, parsed as a JSON object. Quote-aware tokenising means you can paste multi-word string values without escaping each one.

## 💡 Examples

```text
voicetel> inumbering search-inventory --npa=201 --state=NJ --limit=5
{
  "numbers": [
    {"number": "2015550100", "rateCenter": "Newark", "city": "Newark", "province": "NJ", "lata": "224"},
    ...
  ]
}

voicetel> inumbering order {"numbers":[{"Value":"2015550100"}]}
{
  "orderId": "ord-9k2j",
  "amountCharged": 1.00,
  "numbersOrdered": ["2015550100"]
}

voicetel> numbers set-forward 2015550100 {"destination":2125550199}
{"number": "2015550100", "forwardTo": "2125550199"}

voicetel> messaging send {"fromNumber":"2015550100","toNumber":"2125550199","text":"hello from VoiceTel CLI"}
{
  "id": "tx-abc123",
  "type": "sms",
  "fromNumber": "2015550100",
  "toNumber":   "2125550199",
  "parts": 1
}

voicetel> account credits
{
  "credits": [
    {"date": "2026-05-15", "paid": true, "amount": 100.00},
    {"date": "2026-04-15", "paid": true, "amount": 100.00}
  ]
}
```

## 📖 API Documentation

- **Reference docs:** [voicetel.com/docs/api/v2.2/](https://voicetel.com/docs/api/v2.2/)
- **Interactive playground:** [voicetel.com/docs/api/v2.2/playground/](https://voicetel.com/docs/api/v2.2/playground/) — try the API in your browser without writing any code
- **API credentials:** [voicetel.com/docs/api/v2.2/credentials/](https://voicetel.com/docs/api/v2.2/credentials/)
- **Go SDK:** [github.com/voicetel/go-sdk](https://github.com/voicetel/go-sdk)

## 🙌 Contributors

- [Michael Mavroudis](https://github.com/mavroudis) — Lead Developer

Contributions welcome. Open an issue describing the change, or send a pull request against `main`.

## 💖 Sponsors

| Sponsor | Contribution |
|---------|--------------|
| [VoiceTel Communications](https://voicetel.com) | Primary development and production hosting |

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
