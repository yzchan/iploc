ip归属地查询库（基于纯真ip库）
-----

## About

又一个基于纯真ip库的解析程序。初学Golang练手的作品。

## Installation

```shell
go get -u github.com/yzchan/ip-locate
```

## Quickstart

```go
package main

import (
	"fmt"
	iplocate "github.com/yzchan/ip-locate"
)

func main() {
	q, err := iplocate.NewQQWryParser("../data/qqwry.dat")
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
// 测试环境 2017款13寸MBP 8GB(2133MHz)
goos: darwin
goarch: amd64
pkg: github.com/yzchan/ip-locate
cpu: Intel(R) Core(TM) i5-7360U CPU @ 2.30GHz
BenchmarkFind
BenchmarkFind-4                  6399915               552.2 ns/op           568 B/op          6 allocs/op
BenchmarkFindParallel
BenchmarkFindParallel-4         12724434               335.8 ns/op           568 B/op          6 allocs/op
PASS
ok      github.com/yzchan/ip-locate     8.769s
```

## Features

- 纯真ip库的下载
- 将格式化之后的数据缓存在内存中，以获得更高的查询性能。

## Thinks

- [kayon/iploc](https://github.com/kayon/iploc)
- [freshcn/qqwry](https://github.com/freshcn/qqwry)
- [Dnomd343](https://zhuanlan.zhihu.com/p/360624952)
