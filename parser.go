package iplocate

import (
	"encoding/binary"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io/ioutil"
	"net"
	"os"
)

type Record struct {
	RecordA string
	RecordB string
}

type Result struct {
	StartIP net.IP
	StopIP  net.IP
	Record
}

type QQWryParser struct {
	buffers []byte
	len     int
	head    uint32
	tail    uint32
	enc     *encoding.Decoder
	maps    map[uint32]Record
}

func NewQQWryParser(filepath string) (q *QQWryParser, err error) {
	q = &QQWryParser{}

	f, err := os.OpenFile(filepath, os.O_RDONLY, 0400)
	if err != nil {
		return q, err
	}
	defer f.Close()

	buffer, err := ioutil.ReadAll(f)
	if err != nil {
		return q, err
	}

	q.buffers = buffer
	q.head = binary.LittleEndian.Uint32(buffer[:4])
	q.tail = binary.LittleEndian.Uint32(buffer[4:8])
	q.len = int((q.tail-q.head)/7) + 1
	q.enc = simplifiedchinese.GBK.NewDecoder()

	return q, nil
}

// Find 查询函数
func (q *QQWryParser) Find(ipStr string) (recordA string, recordB string) {
	ip := binary.BigEndian.Uint32(net.ParseIP(ipStr).To4())
	if len(q.maps) > 0 {
		return q.findInMap(ip)
	}
	_, _, areaOffset := q.searchIndex(ip)
	return q.readRecords(areaOffset)
}

// Query TODO 标准查询函数，接收 net.IP 类型的参数
func (q *QQWryParser) Query(ip net.IP) (recordA string, recordB string) {
	return
}

// Version 返回版本信息
func (q *QQWryParser) Version() string {
	a, b := q.Find("255.255.255.0")
	return a + b
}

func (q *QQWryParser) FormatMap() {
	q.maps = make(map[uint32]Record, q.len)
	for i := q.head; i <= q.tail; i += 7 {
		recordA, recordB := q.readRecords(q.fillOffset(q.buffers[i+4 : i+7]))
		q.maps[binary.LittleEndian.Uint32(q.buffers[i:i+4])] = Record{recordA, recordB}
	}
}

func (q *QQWryParser) findInMap(ip uint32) (string, string) {
	_, ipu, _ := q.searchIndex(ip)
	r, ok := q.maps[ipu]
	if !ok {
		return "", ""
	}
	return r.RecordA, r.RecordB
}

/**
 * 纯真ip库中有的数据只占用3个byte，这里填充为4byte的uint32
 */
func (q *QQWryParser) fillOffset(b3 []byte) uint32 {
	return uint32(b3[0]) | uint32(b3[1])<<8 | uint32(b3[2])<<16 | 00<<24
}

/**
 * 根据索引区偏移量读取索引区数据 返回起始ip和记录区偏移量
 * 索引区每条索引是一个长度为7的[]byte，前4个byte表示起始ip 后3个byte表示记录区偏移量
 */
func (q *QQWryParser) readIndex(offset uint32) (startIp uint32, recordOffset uint32) {
	ip := q.buffers[offset : offset+4]
	startIp = binary.LittleEndian.Uint32(ip)
	recordOffset = q.fillOffset(q.buffers[offset+4 : offset+7])
	return
}

func (q *QQWryParser) searchIndex(target uint32) (indexOffset uint32, startIp uint32, recordOffset uint32) {
	head := uint32(0)
	tail := (q.tail-q.head)/7 + 1

	mid := (head + tail) / 2
	var ipMid uint32
	for i := 0; ; i++ {
		ipMid, recordOffset = q.readIndex(mid*7 + q.head)
		if head == mid {
			indexOffset = mid*7 + q.head
			startIp = ipMid
			return
		}
		if target < ipMid {
			tail = mid
		} else {
			head = mid
		}
		mid = (head + tail) / 2
	}
}

/**
 * 根据偏移量整体读取记录A和记录B的值
 */
func (q *QQWryParser) readRecords(offset uint32) (textA string, textB string) {
	buff := q.buffers[offset : offset+8]
	//fmt.Printf("%#08x: %#x %#x [%#x][%#x]\n", offset, buff[:4], buff[4:], buff[0], buff[4])
	if buff[4] == 0x01 { //记录模式245
		return q.readRecords(q.fillOffset(buff[5:]) - 4) // 非重定向模式 前4byte为指针 需要后移4位
	}

	var pos2 uint32
	textA, pos2 = q.readRecord(offset + 4)
	textB, _ = q.readRecord(pos2)
	if textB == " CZ88.NET" {
		textB = ""
	}
	return
}

/**
 * 读取记录A/记录B：需传入偏移量。返回记录A/记录B的值，同时返回新的偏移量
 * 因为记录A/记录B存储采用c语言字符数组的方式（遇到\0表示结束）
 * 所以读取记录B需要知道记录A的长度，所以在读取记录A的时候返回新的偏移量供读取记录B使用
 */
func (q *QQWryParser) readRecord(offset uint32) (record string, cursor uint32) {
	cursor = offset
	// 先预读4个byte的内容分析 如果第一个字节是0x02说明是重定向
	b4 := q.buffers[offset : offset+4]

	if b4[0] == 0x02 {
		record, cursor = q.readRecord(q.fillOffset(b4[1:]))
		return record, offset + 4
	}

	// 重新开始读取，遇到\0结束
	for {
		cursor++
		if q.buffers[cursor] == 0x00 {
			break
		}
	}
	buff := q.buffers[offset:cursor]
	record, _ = q.enc.String(string(buff))
	//Record = string(buff)
	cursor++
	return
}
