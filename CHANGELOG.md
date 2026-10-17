# Changelog

## 0.3.0 — 2026-10-17

Track go-sdk v2.2.16 / VoiceTel API spec v2.2.16. **Breaking flag changes.**

### Breaking
- `inumbering search-inventory`: `--tn-mask` (required), `--quantity`, `--province`, `--rate-center`, `--lata` — the old npa/nxx/state/rate-center filters were removed.
- `inumbering coverage`: `--count-by` is now required; filters are `--loc-state`/`--city`/`--rc-abbre`/`--npa`/`--lata`.
- `messaging history`: `--number` is required; `--from`/`--to` replace `--start`/`--end`.
- `messaging campaign-status`, `numbers unassign-campaign`, `numbers bulk-unassign-campaign`, and `acl remove` now take the required ids/campaignId/cidrs arguments.
- `login` treats the username as a string.

### Security
- REPL history (`~/.voicetel/history`) is mode 0600, and lines beginning with `login` or `set api-key` are never persisted.

### Fixed
- README/help examples referenced nonexistent commands (`account numbers`, `account info`, `acl add`); replaced with real ones.
- Rate-limit docs match the spec (4 endpoints at 6/hr/IP + health check at 30/hr/account).

## 0.2.8 and earlier

See git history.
