ip归属地查询库（基于纯真ip库）
-----

## About

纯真ip库Golang解析程序。

## Document

- [纯真ip库的存储格式与解析方式浅析](/doc/parse_qqwry.md)

## Installation

```shell
go get -u github.com/yzchan/iploc
```

## Quickstart

```go
package main

import (
	"fmt"
	"github.com/yzchan/iploc"
)

func main() {
	q, err := iploc.NewQQWryParser("/path/to/file/qqwry.dat")
	if err != nil {
		panic(err)
	}
	textA, textB := q.Find("127.0.0.1")
	fmt.Println(textA, textB)
}
```

## Benchmarks

```shell
go test -v -run="none" -bench=. -benchmem -benchtime=3s
```

```
// 测试环境 2017款13寸MacBookPro
goos: darwin
goarch: amd64
pkg: github.com/yzchan/iploc
cpu: Intel(R) Core(TM) i5-7360U CPU @ 2.30GHz
BenchmarkFind
BenchmarkFind-4                  6399915               552.2 ns/op           568 B/op          6 allocs/op
BenchmarkFindParallel
BenchmarkFindParallel-4         12724434               335.8 ns/op           568 B/op          6 allocs/op
PASS
ok      github.com/yzchan/iploc     8.769s
```

## Thinks

- [纯真(CZ88.net)](https://www.cz88.net/)
- [kayon/iploc](https://github.com/kayon/iploc)
- [freshcn/qqwry](https://github.com/freshcn/qqwry)
- [Dnomd343](https://zhuanlan.zhihu.com/p/360624952)
