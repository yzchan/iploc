package iploc

import (
	"math/rand"
	"testing"
	"time"
)

const filePath = "download/qqwry.dat"

func TestFind(t *testing.T) {
	var results = []struct {
		ip      string
		recordA string
		recordB string
	}{
		{"0.0.0.1", "IANA", "保留地址"},
		{"127.0.0.1", "本机地址", ""},
		{"255.255.255.1", "纯真网络", "2022年04月27日IP数据"},
	}
	q, err := NewQQWryParser(filePath)
	if err != nil {
		t.Fatal("读取ip库文件失败")
	}
	q.FormatMap() // 格式化数据到map
	t.Log("开始测试Find函数")
	errFlag := false
	for index, result := range results {
		recordA, recordB := q.Find(result.ip)
		t.Logf("第[%d]组ip [%s]\n", index, result.ip)
		t.Logf("  |-预期结果：[%s] [%s]\n", result.recordA, result.recordB)
		t.Logf("  |-查询结果：[%s] [%s]\n", recordA, recordB)

		if recordA != result.recordA || recordB != result.recordB {
			errFlag = true
		}
	}
	if errFlag {
		t.Fatal("\x1b[31m测试失败！\x1b[0m")
	}
	t.Log("\x1b[32m测试通过！\x1b[0m")
}

//func BenchmarkFormat(b *testing.B) {
//	b.StopTimer()
//
//	q, err := NewQQWryParser(filePath)
//	if err != nil {
//		panic(err)
//	}
//
//	rand.Seed(time.Now().UnixNano())
//	b.StartTimer()
//	for i := 0; i < b.N; i++ {
//		q.FormatMap()
//	}
//}

func BenchmarkFind(b *testing.B) {
	b.StopTimer()

	q, err := NewQQWryParser(filePath)
	if err != nil {
		panic(err)
	}
	//q.FormatMap()

	rand.Seed(time.Now().UnixNano())
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		q.Find("127.0.0.1")
	}
}

func BenchmarkFindParallel(b *testing.B) {
	b.StopTimer()
	q, err := NewQQWryParser(filePath)
	if err != nil {
		b.Fatal(err)
	}
	//q.FormatMap()
	rand.Seed(time.Now().UnixNano())
	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Find("127.0.0.1")
		}
	})
}
