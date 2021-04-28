package iplocate

import (
	"math/rand"
	"testing"
	"time"
)

func TestFind(t *testing.T) {
	var results = []struct {
		ip      string
		recordA string
		recordB string
	}{
		{"0.0.0.1", "IANA", "保留地址"},
		{"127.0.0.1", "本机地址", " CZ88.NET"},
		{"255.255.255.1", "纯真网络", "2021年04月14日IP数据"},
	}
	q, err := NewQQWryParser("data/qqwry.dat")
	if err != nil {
		t.Fatal("读取到ip库文件失败")
	}
	t.Log("开始测试Find函数")
	errFlag := false
	for index, result := range results {
		recordA, recordB := q.Find(result.ip)
		t.Logf("\x1b[32m 第[%d]组ip [%s] \x1b[0m\n", index, result.ip)
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

func BenchmarkFind(b *testing.B) {
	b.StopTimer()

	q, err := NewQQWryParser("data/qqwry.dat")
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		q.Find("127.0.0.1")
	}
}

func BenchmarkFindParallel(b *testing.B) {
	b.StopTimer()
	q, err := NewQQWryParser("data/qqwry.dat")
	if err != nil {
		b.Fatal(err)
	}
	rand.Seed(time.Now().UnixNano())
	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Find("127.0.0.1")
		}
	})
}
