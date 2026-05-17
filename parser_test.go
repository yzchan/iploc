package iploc

import (
	"encoding/binary"
	"errors"
	"net"
	"os"
	"sync"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestFindAndQuery(t *testing.T) {
	q := newTestParser(t)
	results := []struct {
		ip      string
		recordA string
		recordB string
	}{
		{"0.0.0.1", "IANA", "保留地址"},
		{"1.0.0.1", "Local", ""},
		{"127.0.0.1", "Local", ""},
		{"128.0.0.1", "Version", "Data"},
		{"255.255.255.1", "Version", "Data"},
	}

	assertFindResults(t, q, results)
	assertQueryResults(t, q, results)
}

func TestQueryResult(t *testing.T) {
	q := newTestParser(t)

	result, err := q.QueryResult(net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Fatalf("QueryResult() error = %v", err)
	}
	if result.StartIP.String() != "1.0.0.0" || result.StopIP.String() != "127.255.255.255" {
		t.Fatalf("QueryResult() range = %s-%s, want 1.0.0.0-127.255.255.255", result.StartIP, result.StopIP)
	}
	if result.RecordA != "Local" || result.RecordB != "" {
		t.Fatalf("QueryResult() record = [%s] [%s], want [Local] []", result.RecordA, result.RecordB)
	}
}

func TestFormatMap(t *testing.T) {
	q := newTestParser(t)
	if err := q.FormatMap(); err != nil {
		t.Fatalf("FormatMap() error = %v", err)
	}

	assertFindResults(t, q, []struct {
		ip      string
		recordA string
		recordB string
	}{
		{"0.0.0.1", "IANA", "保留地址"},
		{"127.0.0.1", "Local", ""},
		{"255.255.255.1", "Version", "Data"},
	})
}

func TestQueryResultWithMap(t *testing.T) {
	q := newTestParser(t)
	if err := q.FormatMap(); err != nil {
		t.Fatalf("FormatMap() error = %v", err)
	}

	result, err := q.QueryResult(net.ParseIP("255.255.255.1"))
	if err != nil {
		t.Fatalf("QueryResult() error = %v", err)
	}
	if result.StartIP.String() != "128.0.0.0" || result.StopIP.String() != "255.255.255.255" {
		t.Fatalf("QueryResult() range = %s-%s, want 128.0.0.0-255.255.255.255", result.StartIP, result.StopIP)
	}
	if result.RecordA != "Version" || result.RecordB != "Data" {
		t.Fatalf("QueryResult() record = [%s] [%s], want [Version] [Data]", result.RecordA, result.RecordB)
	}
}

func TestQueryRejectsInvalidIP(t *testing.T) {
	q := newTestParser(t)

	for _, ip := range []string{"bad-ip", "2001:db8::1"} {
		t.Run(ip, func(t *testing.T) {
			_, _, err := q.Query(net.ParseIP(ip))
			if !errors.Is(err, ErrInvalidIP) {
				t.Fatalf("Query() error = %v, want ErrInvalidIP", err)
			}
		})
	}
}

func TestFindInvalidIPDoesNotPanic(t *testing.T) {
	q := newTestParser(t)

	recordA, recordB := q.Find("bad-ip")
	if recordA != "" || recordB != "" {
		t.Fatalf("Find() = [%s] [%s], want empty result", recordA, recordB)
	}
}

func TestNewQQWryParserRejectsInvalidDatabase(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "short header", data: []byte("short")},
		{name: "tail before head", data: buildHeader(32, 24, 40)},
		{name: "unaligned index", data: buildHeader(16, 18, 32)},
		{name: "tail out of range", data: buildHeader(16, 30, 32)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewQQWryParserFromBytes(tt.data)
			if !errors.Is(err, ErrInvalidDatabase) {
				t.Fatalf("NewQQWryParserFromBytes() error = %v, want ErrInvalidDatabase", err)
			}
		})
	}
}

func TestReadRecordsRejectsBadRedirect(t *testing.T) {
	q := newTestParser(t)
	q.buffers[60] = redirectModeAll
	put3(q.buffers[61:64], 3)

	_, _, err := q.Query(net.ParseIP("255.255.255.1"))
	if !errors.Is(err, ErrInvalidDatabase) {
		t.Fatalf("Query() error = %v, want ErrInvalidDatabase", err)
	}
}

func TestQueryIsSafeForConcurrentUse(t *testing.T) {
	q := newTestParser(t)
	var wg sync.WaitGroup

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				recordA, recordB, err := q.Query(net.ParseIP("0.0.0.1"))
				if err != nil || recordA != "IANA" || recordB != "保留地址" {
					t.Errorf("Query() = [%s] [%s], %v", recordA, recordB, err)
					return
				}
			}
		}()
	}

	wg.Wait()
}

func BenchmarkFind(b *testing.B) {
	q := newRealBenchmarkParser(b)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q.Find("127.0.0.1")
	}
}

func BenchmarkFindParallel(b *testing.B) {
	q := newRealBenchmarkParser(b)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Find("127.0.0.1")
		}
	})
}

func BenchmarkFindWithMap(b *testing.B) {
	q := newRealBenchmarkParser(b)
	if err := q.FormatMap(); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q.Find("127.0.0.1")
	}
}

func assertFindResults(t *testing.T, q *QQWryParser, results []struct {
	ip      string
	recordA string
	recordB string
}) {
	t.Helper()

	for _, result := range results {
		recordA, recordB := q.Find(result.ip)
		if recordA != result.recordA || recordB != result.recordB {
			t.Fatalf("Find(%q) = [%s] [%s], want [%s] [%s]",
				result.ip, recordA, recordB, result.recordA, result.recordB)
		}
	}
}

func assertQueryResults(t *testing.T, q *QQWryParser, results []struct {
	ip      string
	recordA string
	recordB string
}) {
	t.Helper()

	for _, result := range results {
		recordA, recordB, err := q.Query(net.ParseIP(result.ip))
		if err != nil {
			t.Fatalf("Query(%q) error = %v", result.ip, err)
		}
		if recordA != result.recordA || recordB != result.recordB {
			t.Fatalf("Query(%q) = [%s] [%s], want [%s] [%s]",
				result.ip, recordA, recordB, result.recordA, result.recordB)
		}
	}
}

func newTestParser(t *testing.T) *QQWryParser {
	t.Helper()

	q, err := NewQQWryParserFromBytes(testQQWryData())
	if err != nil {
		t.Fatalf("NewQQWryParserFromBytes() error = %v", err)
	}
	return q
}

func newRealBenchmarkParser(b *testing.B) *QQWryParser {
	b.Helper()

	const path = "data/qqwry-2021.04.14.dat"
	if _, err := os.Stat(path); err != nil {
		b.Skipf("real benchmark database not available: %v", err)
	}

	q, err := NewQQWryParser(path)
	if err != nil {
		b.Fatalf("NewQQWryParser() error = %v", err)
	}
	return q
}

func testQQWryData() []byte {
	data := make([]byte, 120)
	binary.LittleEndian.PutUint32(data[0:4], 64)
	binary.LittleEndian.PutUint32(data[4:8], 78)

	copy(data[8:12], ipLE(0, 255, 255, 255))
	copyCString(data, 12, []byte("IANA"))
	copyCString(data, 17, mustGBK("保留地址"))

	copy(data[32:36], ipLE(127, 255, 255, 255))
	data[36] = redirectModePart
	put3(data[37:40], 50)
	copyCString(data, 40, []byte(" CZ88.NET"))
	copyCString(data, 50, []byte("Local"))

	copy(data[56:60], ipLE(255, 255, 255, 255))
	data[60] = redirectModeAll
	put3(data[61:64], 90)
	copy(data[86:90], []byte{0, 0, 0, 0})
	copyCString(data, 90, []byte("Version"))
	copyCString(data, 98, []byte("Data"))

	copyIndex(data, 64, ipLE(0, 0, 0, 0), 8)
	copyIndex(data, 71, ipLE(1, 0, 0, 0), 32)
	copyIndex(data, 78, ipLE(128, 0, 0, 0), 56)

	return data
}

func buildHeader(head, tail uint32, size int) []byte {
	data := make([]byte, size)
	binary.LittleEndian.PutUint32(data[0:4], head)
	binary.LittleEndian.PutUint32(data[4:8], tail)
	return data
}

func copyIndex(data []byte, offset int, ip []byte, recordOffset uint32) {
	copy(data[offset:offset+4], ip)
	put3(data[offset+4:offset+7], recordOffset)
}

func copyCString(data []byte, offset int, value []byte) {
	copy(data[offset:], value)
	data[offset+len(value)] = 0
}

func mustGBK(value string) []byte {
	data, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(value))
	if err != nil {
		panic(err)
	}
	return data
}

func ipLE(a, b, c, d byte) []byte {
	return []byte{d, c, b, a}
}

func put3(data []byte, value uint32) {
	data[0] = byte(value)
	data[1] = byte(value >> 8)
	data[2] = byte(value >> 16)
}
