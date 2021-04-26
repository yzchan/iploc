package iplocate

import (
	"math/rand"
	"testing"
	"time"
)

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
