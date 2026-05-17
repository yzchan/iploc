# iploc

[![CI](https://github.com/yzchan/iploc/actions/workflows/ci.yml/badge.svg)](https://github.com/yzchan/iploc/actions/workflows/ci.yml)

A Go library and CLI for querying IPv4 locations from QQWry / 纯真 IP database files. IPv6 is not supported.

`iploc` loads a QQWry `.dat` file into memory, searches the IPv4 index with binary search, resolves QQWry record redirects, and converts GBK record text to UTF-8. It is designed for Go applications, shell scripts, and agent/AI tool integrations that need deterministic local IP-location lookup.

## Features

- Pure Go QQWry parser for IPv4 location records.
- IPv4-only by design; IPv6 input returns `ErrInvalidIP`.
- Error-aware API via `Query(net.IP)`.
- Backward-compatible `Find(string)` API for existing callers.
- Optional `FormatMap()` preload mode for faster repeated lookups.
- CLI query tool with text, JSON, and JSONL output.
- Stdin and multi-IP input for scripts and agent workflows.
- Fixture-based tests that do not depend on a specific QQWry data release.

## Data Files

The repository includes `data/qqwry-2021.04.14.dat` as a convenience sample for local testing and CLI demos because QQWry data can be difficult to obtain from public sources.

The sample data may be outdated and should not be treated as authoritative production data. For production use, provide your own QQWry `.dat` file and pass its path to the Go API or CLI with `--db`.

QQWry data rights belong to their respective owners. This project focuses on parsing and querying compatible `.dat` files.

## Installation

Use the library from Go code:

```shell
go get github.com/yzchan/iploc
```

Install the CLI with Go:

```shell
go install github.com/yzchan/iploc/cmd/iploc@latest
```

Run the CLI from source:

```shell
go run ./cmd/iploc --help
```

Or build a local binary:

```shell
go build -o iploc ./cmd/iploc
```

## Go Usage

```go
package main

import (
	"errors"
	"fmt"
	"net"

	"github.com/yzchan/iploc"
)

func main() {
	q, err := iploc.NewQQWryParser("/path/to/qqwry.dat")
	if err != nil {
		panic(err)
	}

	recordA, recordB, err := q.Query(net.ParseIP("127.0.0.1"))
	if errors.Is(err, iploc.ErrInvalidIP) {
		panic("only IPv4 addresses are supported")
	}
	if err != nil {
		panic(err)
	}

	fmt.Println(recordA, recordB)
}
```

For existing code, `Find(string)` remains available and returns empty strings when the query would fail. New code should prefer `Query(net.IP)` so invalid input and database errors can be handled explicitly.

## CLI Usage

Query one IP:

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat 127.0.0.1
```

Query multiple IPs:

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat 0.0.0.1 127.0.0.1
```

Read IPs from stdin:

```shell
printf '0.0.0.1\n127.0.0.1\n' | go run ./cmd/iploc --db data/qqwry-2021.04.14.dat
```

Emit JSONL for agent and CLI toolchains:

```shell
printf '0.0.0.1\nbad-ip\n' | go run ./cmd/iploc --db data/qqwry-2021.04.14.dat --format jsonl
```

Emit a JSON array:

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat --format json 0.0.0.1 127.0.0.1
```

Useful flags:

- `--format text|json|jsonl` controls output format; default is `text`.
- `--map` preloads records into a map before querying.
- `--fail-on-error` returns a non-zero exit code if any IP query fails.
- `--version` prints the CLI version and exits.

## API Overview

- `NewQQWryParser(path)` loads a `.dat` file into memory.
- `NewQQWryParserFromBytes(data)` creates a parser from bytes and copies the input.
- `Query(ip)` returns record A, record B, and an error.
- `QueryResult(ip)` returns the matched start IP, stop IP, record A, record B, and an error.
- `Find(ipString)` keeps compatibility with older releases.
- `FormatMap()` pre-parses records into a map for faster repeated lookups.
- `VersionWithError()` / `Version()` reads the QQWry version record.

Sentinel errors:

- `ErrInvalidIP`
- `ErrInvalidDatabase`
- `ErrNilParser`

## Benchmarks

Benchmarks use the repository sample database at `data/qqwry-2021.04.14.dat`. Run them with:

```shell
go test -run='^$' -bench=. -benchmem -benchtime=3s
```

Latest local result:

```text
// Apple M4 Pro, Go 1.23.3, module minimum Go version 1.20
// Database: data/qqwry-2021.04.14.dat
goos: darwin
goarch: arm64
pkg: github.com/yzchan/iploc
cpu: Apple M4 Pro
BenchmarkFind-12             11545689        292.1 ns/op      600 B/op      7 allocs/op
BenchmarkFindParallel-12     46296171         84.39 ns/op     600 B/op      7 allocs/op
BenchmarkFindWithMap-12      28503196        123.4 ns/op       13 B/op      0 allocs/op
PASS
ok      github.com/yzchan/iploc     16.482s
```

The included sample database is useful for repeatable local measurements, but production performance still depends on your actual QQWry data file and query mix.

## Releases

Tagged releases build cross-platform CLI binaries with GitHub Actions. The release workflow injects the tag into `iploc --version`.

Example local versioned build:

```shell
go build -ldflags "-X main.version=v0.0.0-dev" -o bin/iploc ./cmd/iploc
./bin/iploc --version
```

## Documentation

- [QQWry format and parser notes](docs/qqwry-format.md)
- [Changelog](CHANGELOG.md)

## Development

```shell
go test ./...
go vet ./...
go test -race ./...
```

The CI workflow runs test, vet, and race checks on supported Go versions.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

## Related Links

- [纯真(CZ88.net)](https://www.cz88.net/)
- [kayon/iploc](https://github.com/kayon/iploc)
- [freshcn/qqwry](https://github.com/freshcn/qqwry)
- [Dnomd343](https://zhuanlan.zhihu.com/p/360624952)
