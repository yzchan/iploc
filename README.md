# iploc

[![CI](https://github.com/yzchan/iploc/actions/workflows/ci.yml/badge.svg)](https://github.com/yzchan/iploc/actions/workflows/ci.yml)

`iploc` 是一个用于查询旧版 QQWry / 纯真 IP `.dat` 数据库的 Go 库和命令行工具。项目仅支持 IPv4，不支持 IPv6。

> [!IMPORTANT]
> 本项目只针对旧版 QQWry / 纯真 IP `.dat` 格式。新版 CZ88 / 纯真 IP 数据库已经改用 `.czdb` 格式，和本项目当前解析器不兼容。如果你需要查询新版 CZDB 数据，建议使用 [tagphi/czdb-search-golang](https://github.com/tagphi/czdb-search-golang)。本仓库主要作为旧版 `.dat` 数据库的历史兼容项目保留。

`iploc` 会把 QQWry `.dat` 文件加载到内存中，通过二分查找定位 IPv4 索引，解析 QQWry 记录区跳转，并把 GBK 编码的地区文本转换为 UTF-8。它适用于仍然需要查询旧版纯真 IP 库的 Go 程序、Shell 脚本和本地 CLI 集成场景。

## 功能特性

- 纯 Go 实现的 QQWry IPv4 记录解析器。
- 仅支持 IPv4；IPv6 或非法 IP 会返回 `ErrInvalidIP`。
- 提供带错误返回的 `Query(net.IP)` API。
- 保留兼容旧代码的 `Find(string)` API。
- 支持 `FormatMap()` 预加载模式，用于提升重复查询性能。
- 提供 `cmd/iploc` 命令行工具。
- CLI 支持文本、JSON、JSONL 输出。
- CLI 支持 stdin、多 IP 输入，方便脚本和 Agent 工具调用。
- 测试使用小型 fixture，不依赖某个特定的 QQWry 数据版本。

## 数据文件

仓库包含 `data/qqwry-2021.04.14.dat`，主要用于本地测试和 CLI 示例。由于旧版纯真 IP `.dat` 文件现在不太容易从公开渠道获取，所以这里保留一份样例数据，方便调试。

需要注意：该样例数据已经比较旧，不应作为生产环境的权威数据。生产环境请自行提供需要使用的 QQWry `.dat` 文件，并通过 Go API 或 CLI 的 `--db` 参数指定路径。

QQWry / 纯真 IP 数据版权归其对应权利方所有。本项目只关注兼容格式的解析和查询。

## 安装

在 Go 项目中使用：

```shell
go get github.com/yzchan/iploc
```

安装 CLI：

```shell
go install github.com/yzchan/iploc/cmd/iploc@latest
```

从源码运行 CLI：

```shell
go run ./cmd/iploc --help
```

构建本地二进制：

```shell
go build -o iploc ./cmd/iploc
```

## Go 使用示例

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

`Find(string)` 会继续保留，用于兼容已有代码；当查询失败时它会返回空字符串。新代码建议使用 `Query(net.IP)`，这样可以显式处理非法 IP 和数据库错误。

## CLI 使用示例

查询单个 IP：

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat 127.0.0.1
```

查询多个 IP：

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat 0.0.0.1 127.0.0.1
```

从 stdin 读取 IP：

```shell
printf '0.0.0.1\n127.0.0.1\n' | go run ./cmd/iploc --db data/qqwry-2021.04.14.dat
```

输出 JSONL，方便脚本或 Agent 工具链处理：

```shell
printf '0.0.0.1\nbad-ip\n' | go run ./cmd/iploc --db data/qqwry-2021.04.14.dat --format jsonl
```

输出 JSON 数组：

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat --format json 0.0.0.1 127.0.0.1
```

常用参数：

- `--format text|json|jsonl`：输出格式，默认是 `text`。
- `--map`：查询前把记录预加载到 map 中。
- `--fail-on-error`：只要有任意 IP 查询失败，就返回非零退出码。
- `--version`：输出 CLI 版本号并退出。

## API 概览

- `NewQQWryParser(path)`：从 `.dat` 文件加载数据库。
- `NewQQWryParserFromBytes(data)`：从字节数据创建 parser，并复制输入数据。
- `Query(ip)`：返回记录 A、记录 B 和错误。
- `QueryResult(ip)`：返回命中的起始 IP、结束 IP、记录 A、记录 B 和错误。
- `Find(ipString)`：兼容旧版本 API。
- `FormatMap()`：提前解析记录并写入 map，用于提升重复查询性能。
- `VersionWithError()` / `Version()`：读取 QQWry 数据库版本记录。

错误类型：

- `ErrInvalidIP`
- `ErrInvalidDatabase`
- `ErrNilParser`

## Benchmark

Benchmark 使用仓库中的样例数据库 `data/qqwry-2021.04.14.dat`。运行方式：

```shell
go test -run='^$' -bench=. -benchmem -benchtime=3s
```

最近一次本地结果：

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

样例数据库适合做可重复的本地测试，但生产性能仍然取决于实际使用的数据文件和查询模式。

## 文档

- [QQWry 格式和解析说明](docs/qqwry-format.md)
- [更新日志](CHANGELOG.md)

## 相关链接

- [纯真 CZ88.net](https://www.cz88.net/)
- [tagphi/czdb-search-golang](https://github.com/tagphi/czdb-search-golang)
- [kayon/iploc](https://github.com/kayon/iploc)
- [freshcn/qqwry](https://github.com/freshcn/qqwry)
- [Dnomd343](https://zhuanlan.zhihu.com/p/360624952)
