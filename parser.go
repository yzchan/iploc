package iplocate

import (
	"encoding/binary"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io/ioutil"
	"net"
	"os"
)

type QQWryParser struct {
	buffers   []byte
	indexHead uint32
	indexTail uint32
	Length    int
	decoder   *encoding.Decoder
	//maps      map[uint32]string
}

func NewQQWryParser(filepath string) (q QQWryParser, err error) {
	q = QQWryParser{}

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
	q.indexHead = binary.LittleEndian.Uint32(buffer[:4])
	q.indexTail = binary.LittleEndian.Uint32(buffer[4:8])
	q.Length = int((q.indexTail-q.indexHead)/7) + 1
	//q.maps = make(map[uint32]string, q.Length)
	q.decoder = simplifiedchinese.GBK.NewDecoder()

	return q, nil

}

// 查询函数 解析IP值->查询该ip所属区段和偏移量->根据偏移量查询结果
func (q *QQWryParser) Find(ipStr string) (string, string) {
	ip := net.ParseIP(ipStr)

	ip = ip.To4()
	ipu := binary.BigEndian.Uint32(ip)

	//ip := binary.BigEndian.Uint32(net.ParseIP(ipStr))
	_, _, areaOffset := q.searchIndex(ipu)
	return q.readRecords(areaOffset)
}

/**
 * 根据索引区偏移量读取索引区数据 返回起始ip和记录区偏移量
 * 索引区每条索引是一个长度为7的[]byte，前4个byte表示起始ip 后3个byte表示记录区偏移量
 */
func (q *QQWryParser) readIndex(offset uint32) (startIp uint32, recordOffset uint32) {
	ip := q.buffers[offset : offset+4]
	startIp = binary.LittleEndian.Uint32(ip)
	recordOffset = fillPos(q.buffers[offset+4 : offset+7])
	return
}

func (q *QQWryParser) searchIndex(target uint32) (indexOffset uint32, startIp uint32, recordOffset uint32) {
	head := uint32(0)
	tail := (q.indexTail-q.indexHead)/7 + 1

	mid := (head + tail) / 2
	//fmt.Println(head, mid, tail)
	var ipMid uint32
	for i := 0; ; i++ {
		//fmt.Println("二分查找", i, head, mid, tail)
		//ipMid := q.readIp(mid*7 + q.indexHead)
		ipMid, recordOffset = q.readIndex(mid*7 + q.indexHead)
		if head == mid {
			indexOffset = mid*7 + q.indexHead
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
		return q.readRecords(fillPos(buff[5:]) - 4) // 非重定向模式 前4byte为指针 需要后移4位
	}

	var pos2 uint32
	textA, pos2 = q.readRecord(offset + 4)
	textB, _ = q.readRecord(pos2)
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
		record, cursor = q.readRecord(fillPos(b4[1:]))
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
	record, _ = q.decoder.String(string(buff))
	cursor++
	return
}

/**
 * 纯真ip库中有的数据只占用3个byte，这里处理为4byte的uint32
 */
func fillPos(off3 []byte) uint32 {
	// offset是3byte的偏移量 需要先处理成uint32
	off4 := make([]byte, 4)
	copy(off4, off3)
	offInt := binary.LittleEndian.Uint32(off4)

	return offInt
}
