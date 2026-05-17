# Changelog

All notable changes to this project are documented in this file.

This project follows the spirit of [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and uses semantic versioning for release planning.

## Unreleased

### Added

- Added `Query(net.IP) (string, string, error)` for error-aware IPv4 lookups.
- Added `VersionWithError()` for error-aware database version lookups.
- Added `NewQQWryParserFromBytes` for embedded data, tests, and custom loading flows.
- Added exported sentinel errors: `ErrInvalidIP`, `ErrInvalidDatabase`, and `ErrNilParser`.
- Added package documentation and public API GoDoc comments.
- Added fixture-based tests that do not depend on a full QQWry data file.
- Added GitHub Actions CI for test, vet, and race-test checks across supported Go versions.
- Added `cmd/iploc` for CLI-based IP lookups with text, JSON, and JSONL output.
- Added `QueryResult(net.IP) (Result, error)` for structured range-aware lookups.
- Added CLI `--version` support for scripts and release binaries.
- Added release workflow for cross-platform CLI binaries.
- Added `data/README.md` to document bundled sample data scope and limitations.

### Changed

- `FormatMap()` now returns an `error` instead of panicking on malformed data.
- `Find(string)` remains source-compatible but now delegates to `Query` and returns empty strings on invalid input instead of panicking.
- `NewQQWryParserFromBytes` copies caller-provided bytes to keep parser state immutable from the caller side.
- GBK decoding now uses a fresh decoder per record conversion, making concurrent queries safer.
- Replaced deprecated `ioutil` usage with `os` / `io` helpers.
- Removed the QQWry downloader because current data distribution is better handled outside this library.

### Fixed

- Prevented panics on invalid IPv4 strings, IPv6 input, truncated headers, bad index ranges, malformed offsets, bad redirects, and unterminated records.
- Removed test coupling to a specific bundled QQWry database version.
- Added CLI support for multiple IP arguments, stdin input, JSON/JSONL output, strict failure mode, and structured JSON fields for matched IP ranges.

### Compatibility Notes

- Existing `Find` callers can continue compiling without changes.
- Callers using `FormatMap()` must now handle its returned `error`.
- New code should prefer `Query` over `Find` so invalid input and database corruption can be handled explicitly.
- Users should provide QQWry `.dat` files themselves; the project no longer downloads them.
- The parser and CLI are IPv4-only; IPv6 input is rejected with `ErrInvalidIP`.
