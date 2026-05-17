package iploc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"net"
	"os"
)

const (
	headerSize       = 8
	indexSize        = 7
	redirectModeAll  = 0x01
	redirectModePart = 0x02
	maxRedirectDepth = 8
)

var (
	// ErrInvalidIP indicates that a query input cannot be represented as IPv4.
	ErrInvalidIP = errors.New("iploc: invalid IPv4 address")

	// ErrInvalidDatabase indicates that the QQWry database is truncated or malformed.
	ErrInvalidDatabase = errors.New("iploc: invalid qqwry database")

	// ErrNilParser indicates that a method was called on a nil parser.
	ErrNilParser = errors.New("iploc: nil parser")
)

// Record contains the two text fields stored in a QQWry location record.
type Record struct {
	RecordA string
	RecordB string
}

// Result describes a matched IP range and its associated QQWry record.
type Result struct {
	StartIP net.IP
	StopIP  net.IP
	Record
}

// QQWryParser is an in-memory parser for QQWry IPv4 databases.
type QQWryParser struct {
	buffers []byte
	len     int
	head    uint32
	tail    uint32
	maps    map[uint32]Record
}

// NewQQWryParser loads a QQWry database file into memory.
func NewQQWryParser(filepath string) (q *QQWryParser, err error) {
	buffer, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return newQQWryParser(buffer)
}

// NewQQWryParserFromBytes builds a parser from QQWry database bytes.
// The input is copied so later caller-side mutations do not affect queries.
func NewQQWryParserFromBytes(buffer []byte) (*QQWryParser, error) {
	data := append([]byte(nil), buffer...)
	return newQQWryParser(data)
}

func newQQWryParser(buffer []byte) (*QQWryParser, error) {
	if len(buffer) < headerSize {
		return nil, fmt.Errorf("%w: file is shorter than header", ErrInvalidDatabase)
	}

	q := &QQWryParser{
		buffers: buffer,
		head:    binary.LittleEndian.Uint32(buffer[:4]),
		tail:    binary.LittleEndian.Uint32(buffer[4:8]),
	}

	if q.head < headerSize {
		return nil, fmt.Errorf("%w: index head %d before header", ErrInvalidDatabase, q.head)
	}
	if q.tail < q.head {
		return nil, fmt.Errorf("%w: index tail %d before head %d", ErrInvalidDatabase, q.tail, q.head)
	}
	if (q.tail-q.head)%indexSize != 0 {
		return nil, fmt.Errorf("%w: index range is not aligned", ErrInvalidDatabase)
	}
	if _, err := q.readAt(q.tail, indexSize); err != nil {
		return nil, err
	}
	q.len = int((q.tail-q.head)/indexSize) + 1
	return q, nil
}

// Find queries an IPv4 string and returns record fields.
// It is kept for backward compatibility; use Query in new code to receive errors.
func (q *QQWryParser) Find(ipStr string) (recordA string, recordB string) {
	recordA, recordB, _ = q.Query(net.ParseIP(ipStr))
	return
}

// Query looks up an IPv4 address and returns the matching QQWry record fields.
func (q *QQWryParser) Query(ip net.IP) (recordA string, recordB string, err error) {
	if q == nil {
		return "", "", ErrNilParser
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "", "", ErrInvalidIP
	}

	ipValue := binary.BigEndian.Uint32(ip4)
	if len(q.maps) > 0 {
		return q.findInMap(ipValue)
	}

	_, _, areaOffset, err := q.searchIndex(ipValue)
	if err != nil {
		return "", "", err
	}
	return q.readRecords(areaOffset, 0)
}

// QueryResult looks up an IPv4 address and returns the matching IP range and record.
func (q *QQWryParser) QueryResult(ip net.IP) (Result, error) {
	if q == nil {
		return Result{}, ErrNilParser
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return Result{}, ErrInvalidIP
	}

	ipValue := binary.BigEndian.Uint32(ip4)
	_, startIP, areaOffset, err := q.searchIndex(ipValue)
	if err != nil {
		return Result{}, err
	}
	stopIP, err := q.readStopIP(areaOffset)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		StartIP: uint32ToIP(startIP),
		StopIP:  uint32ToIP(stopIP),
	}
	if len(q.maps) > 0 {
		record, ok := q.maps[startIP]
		if !ok {
			return Result{}, fmt.Errorf("%w: map record missing for %d", ErrInvalidDatabase, startIP)
		}
		result.Record = record
		return result, nil
	}

	recordA, recordB, err := q.readRecords(areaOffset, 0)
	if err != nil {
		return Result{}, err
	}
	result.Record = Record{RecordA: recordA, RecordB: recordB}
	return result, nil
}

// Version returns the QQWry database version text.
// It is kept for backward compatibility; use VersionWithError in new code.
func (q *QQWryParser) Version() string {
	a, b, _ := q.VersionWithError()
	return a + b
}

// VersionWithError returns the QQWry database version record and any lookup error.
func (q *QQWryParser) VersionWithError() (string, string, error) {
	return q.Query(net.ParseIP("255.255.255.0"))
}

// FormatMap preloads parsed records into a map for faster later lookups.
// Call it during initialization before serving concurrent queries.
func (q *QQWryParser) FormatMap() error {
	if q == nil {
		return ErrNilParser
	}
	records := make(map[uint32]Record, q.len)
	for i := q.head; i <= q.tail; i += 7 {
		startIP, recordOffset, err := q.readIndex(i)
		if err != nil {
			return err
		}
		recordA, recordB, err := q.readRecords(recordOffset, 0)
		if err != nil {
			return err
		}
		records[startIP] = Record{recordA, recordB}
	}
	q.maps = records
	return nil
}

func (q *QQWryParser) findInMap(ip uint32) (string, string, error) {
	_, ipu, _, err := q.searchIndex(ip)
	if err != nil {
		return "", "", err
	}
	r, ok := q.maps[ipu]
	if !ok {
		return "", "", fmt.Errorf("%w: map record missing for %d", ErrInvalidDatabase, ipu)
	}
	return r.RecordA, r.RecordB, nil
}

func (q *QQWryParser) readStopIP(offset uint32) (uint32, error) {
	buff, err := q.readAt(offset, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buff), nil
}

func uint32ToIP(ip uint32) net.IP {
	buff := make(net.IP, net.IPv4len)
	binary.BigEndian.PutUint32(buff, ip)
	return buff
}

// 纯真ip库中有的数据只占用3个byte，这里填充为4byte的uint32。
func fillOffset(b3 []byte) (uint32, error) {
	if len(b3) < 3 {
		return 0, fmt.Errorf("%w: 3-byte offset is truncated", ErrInvalidDatabase)
	}
	return uint32(b3[0]) | uint32(b3[1])<<8 | uint32(b3[2])<<16, nil
}

// 根据索引区偏移量读取索引区数据，返回起始ip和记录区偏移量。
func (q *QQWryParser) readIndex(offset uint32) (startIP uint32, recordOffset uint32, err error) {
	index, err := q.readAt(offset, indexSize)
	if err != nil {
		return 0, 0, err
	}
	recordOffset, err = fillOffset(index[4:7])
	if err != nil {
		return 0, 0, err
	}
	return binary.LittleEndian.Uint32(index[:4]), recordOffset, nil
}

func (q *QQWryParser) searchIndex(target uint32) (indexOffset uint32, startIP uint32, recordOffset uint32, err error) {
	if q == nil {
		return 0, 0, 0, ErrNilParser
	}
	if q.len <= 0 {
		return 0, 0, 0, fmt.Errorf("%w: empty index", ErrInvalidDatabase)
	}

	head := uint32(0)
	tail := uint32(q.len)
	for head+1 < tail {
		mid := (head + tail) / 2
		ipMid, _, err := q.readIndex(q.head + mid*indexSize)
		if err != nil {
			return 0, 0, 0, err
		}
		if target < ipMid {
			tail = mid
		} else {
			head = mid
		}
	}
	indexOffset = q.head + head*indexSize
	startIP, recordOffset, err = q.readIndex(indexOffset)
	return
}

// 根据偏移量整体读取记录A和记录B的值。
func (q *QQWryParser) readRecords(offset uint32, depth int) (textA string, textB string, err error) {
	if depth > maxRedirectDepth {
		return "", "", fmt.Errorf("%w: redirect depth exceeded", ErrInvalidDatabase)
	}

	buff, err := q.readAt(offset, headerSize)
	if err != nil {
		return "", "", err
	}
	if buff[4] == redirectModeAll {
		nextOffset, err := fillOffset(buff[5:8])
		if err != nil {
			return "", "", err
		}
		if nextOffset < 4 {
			return "", "", fmt.Errorf("%w: invalid redirect offset %d", ErrInvalidDatabase, nextOffset)
		}
		return q.readRecords(nextOffset-4, depth+1)
	}

	var pos2 uint32
	textA, pos2, err = q.readRecord(offset+4, depth)
	if err != nil {
		return "", "", err
	}
	textB, _, err = q.readRecord(pos2, depth)
	if err != nil {
		return "", "", err
	}
	if textB == " CZ88.NET" {
		textB = ""
	}
	return
}

// 读取记录A/记录B：需传入偏移量。返回记录值，同时返回新的偏移量。
func (q *QQWryParser) readRecord(offset uint32, depth int) (record string, cursor uint32, err error) {
	if depth > maxRedirectDepth {
		return "", 0, fmt.Errorf("%w: redirect depth exceeded", ErrInvalidDatabase)
	}

	cursor = offset
	b4, err := q.readAt(offset, 4)
	if err != nil {
		return "", 0, err
	}

	if b4[0] == redirectModePart {
		nextOffset, err := fillOffset(b4[1:4])
		if err != nil {
			return "", 0, err
		}
		record, _, err = q.readRecord(nextOffset, depth+1)
		return record, offset + 4, err
	}

	for {
		b, err := q.byteAt(cursor)
		if err != nil {
			return "", 0, fmt.Errorf("%w: unterminated record at offset %d", ErrInvalidDatabase, offset)
		}
		if b == 0x00 {
			break
		}
		cursor++
	}
	buff, err := q.readAt(offset, int(cursor-offset))
	if err != nil {
		return "", 0, err
	}
	record, err = simplifiedchinese.GBK.NewDecoder().String(string(buff))
	if err != nil {
		return "", 0, err
	}
	return record, cursor + 1, nil
}

func (q *QQWryParser) readAt(offset uint32, length int) ([]byte, error) {
	if q == nil {
		return nil, ErrNilParser
	}
	if length < 0 {
		return nil, fmt.Errorf("%w: negative read length", ErrInvalidDatabase)
	}
	end := uint64(offset) + uint64(length)
	if end > uint64(len(q.buffers)) {
		return nil, fmt.Errorf("%w: offset %d length %d exceeds file size %d", ErrInvalidDatabase, offset, length, len(q.buffers))
	}
	start := int(offset)
	return q.buffers[start : start+length], nil
}

func (q *QQWryParser) byteAt(offset uint32) (byte, error) {
	buff, err := q.readAt(offset, 1)
	if err != nil {
		return 0, err
	}
	return buff[0], nil
}
