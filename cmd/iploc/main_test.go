package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestRunTextOutput(t *testing.T) {
	dbPath := writeTestDB(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(&stdout, &stderr, cliConfig{
		dbPath: dbPath,
		format: "text",
		ips:    []string{"0.0.0.1", "127.0.0.1"},
	})
	if err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}

	want := "0.0.0.1\tIANA\t保留地址\n127.0.0.1\tLocal\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunJSONLOutputIncludesErrors(t *testing.T) {
	dbPath := writeTestDB(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(&stdout, &stderr, cliConfig{
		dbPath: dbPath,
		format: "jsonl",
		ips:    []string{"bad-ip", "128.0.0.1"},
	})
	if err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl line count = %d, want 2: %q", len(lines), stdout.String())
	}

	var first queryResult
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	if first.IP != "bad-ip" || first.Error == "" {
		t.Fatalf("first result = %+v, want invalid-ip error", first)
	}

	var second queryResult
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("unmarshal second line: %v", err)
	}
	if second.IP != "128.0.0.1" || second.StartIP != "128.0.0.0" || second.StopIP != "255.255.255.255" || second.RecordA != "Version" || second.RecordB != "Data" || second.Error != "" {
		t.Fatalf("second result = %+v, want successful lookup", second)
	}
}

func TestRunVersion(t *testing.T) {
	oldVersion := version
	version = "test-version"
	t.Cleanup(func() { version = oldVersion })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(&stdout, &stderr, cliConfig{showVersion: true})
	if err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}
	if stdout.String() != "iploc test-version\n" {
		t.Fatalf("stdout = %q, want version output", stdout.String())
	}
}

func TestRunFailOnError(t *testing.T) {
	dbPath := writeTestDB(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(&stdout, &stderr, cliConfig{
		dbPath:      dbPath,
		format:      "jsonl",
		failOnError: true,
		ips:         []string{"bad-ip"},
	})
	if err == nil {
		t.Fatal("run() error = nil, want error")
	}
}

func writeTestDB(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "qqwry.dat")
	if err := os.WriteFile(path, cliTestQQWryData(), 0644); err != nil {
		t.Fatalf("write test db: %v", err)
	}
	return path
}

func cliTestQQWryData() []byte {
	data := make([]byte, 120)
	binary.LittleEndian.PutUint32(data[0:4], 64)
	binary.LittleEndian.PutUint32(data[4:8], 78)

	copy(data[8:12], cliIPLE(0, 255, 255, 255))
	cliCopyCString(data, 12, []byte("IANA"))
	cliCopyCString(data, 17, cliMustGBK("保留地址"))

	copy(data[32:36], cliIPLE(127, 255, 255, 255))
	data[36] = 0x02
	cliPut3(data[37:40], 50)
	cliCopyCString(data, 40, []byte(" CZ88.NET"))
	cliCopyCString(data, 50, []byte("Local"))

	copy(data[56:60], cliIPLE(255, 255, 255, 255))
	data[60] = 0x01
	cliPut3(data[61:64], 90)
	copy(data[86:90], []byte{0, 0, 0, 0})
	cliCopyCString(data, 90, []byte("Version"))
	cliCopyCString(data, 98, []byte("Data"))

	cliCopyIndex(data, 64, cliIPLE(0, 0, 0, 0), 8)
	cliCopyIndex(data, 71, cliIPLE(1, 0, 0, 0), 32)
	cliCopyIndex(data, 78, cliIPLE(128, 0, 0, 0), 56)

	return data
}

func cliCopyIndex(data []byte, offset int, ip []byte, recordOffset uint32) {
	copy(data[offset:offset+4], ip)
	cliPut3(data[offset+4:offset+7], recordOffset)
}

func cliCopyCString(data []byte, offset int, value []byte) {
	copy(data[offset:], value)
	data[offset+len(value)] = 0
}

func cliMustGBK(value string) []byte {
	data, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(value))
	if err != nil {
		panic(err)
	}
	return data
}

func cliIPLE(a, b, c, d byte) []byte {
	return []byte{d, c, b, a}
}

func cliPut3(data []byte, value uint32) {
	data[0] = byte(value)
	data[1] = byte(value >> 8)
	data[2] = byte(value >> 16)
}
