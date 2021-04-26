package iplocate

import (
	"encoding/binary"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io/ioutil"
	"math"
	"net"
	"os"
)

type QQWryParser struct {
	buffers   []byte
	indexHead uint32
	indexTail uint32
	Length    int
	//maps      map[uint32]string
}

func (q *QQWryParser) readIp(offset uint32) uint32 {
	ip := q.buffers[offset : offset+4]
	return binary.LittleEndian.Uint32(ip)
}

func (q *QQWryParser) searchIndex(target uint32) (offset uint32, ip uint32, textOffset uint32) {
	head := q.indexHead
	tail := q.indexTail

	//ip1 := q.readIp(head)
	//ip2 := q.readIp(tail)

	mid := uint32(math.Ceil(float64((int(head/7) + int(tail/7)) / 2)))

	for i := 0; ; i++ {
		//fmt.Println("二分查找...", head, mid*7, tail, i)

		ipMid := q.readIp(mid * 7)
		if head == mid*7 {
			return mid * 7, q.readIp(mid * 7), q.readPos(mid * 7)
		}
		if target < ipMid {
			tail = mid * 7
		} else {
			head = mid * 7
		}
		mid = uint32(math.Ceil(float64((int(head/7) + int(tail/7)) / 2)))
	}
}

func (q *QQWryParser) readPos(offset uint32) uint32 {
	return fillPos(q.buffers[offset+4 : offset+7])
}

func (q *QQWryParser) readTexts(offset uint32) (textA string, textB string) {
	buff := q.buffers[offset : offset+8]
	//fmt.Printf("%#08x: %#x %#x [%#x][%#x]\n", offset, buff[:4], buff[4:], buff[0], buff[4])
	if buff[4] == 0x01 { //记录模式245
		return q.readTexts(fillPos(buff[5:]) - 4) // 非重定向模式 前4byte为指针 需要后移4位
	}

	var pos2 uint32
	textA, pos2 = q.readText(offset + 4)
	textB, _ = q.readText(pos2)
	return
}

func (q *QQWryParser) readText(offset uint32) (text string, cursor uint32) {
	cursor = offset
	// 先预读4个byte的内容分析 如果第一个字节是0x02说明是重定向
	//b4 := make([]byte, 4)
	//_, _ = f.ReadAt(b4, int64(cursor))
	b4 := q.buffers[offset : offset+4]

	if b4[0] == 0x02 {
		text, cursor = q.readText(fillPos(b4[1:]))
		return text, offset + 4
	}

	// 重新开始读取
	for {
		cursor++
		if q.buffers[cursor] == 0x00 {
			break
		}
	}
	buff := q.buffers[offset:cursor]
	enc := simplifiedchinese.GBK.NewDecoder()
	text, _ = enc.String(string(buff))
	//text = string(buff)
	cursor++
	return
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

	return q, nil

}

func (q *QQWryParser) Find(ipStr string) (textA string, textB string) {
	ipu := Ip2long(ipStr)

	_, _, areaOffset := q.searchIndex(ipu)
	//areaOffset := q.readPos(index)
	return q.readTexts(areaOffset)
}

func fillPos(off3 []byte) uint32 {
	// offset是3byte的偏移量 需要先处理成uint32
	off4 := make([]byte, 4)
	copy(off4, off3)
	offInt := binary.LittleEndian.Uint32(off4)

	return offInt
}

func Ip2long(ipstr string) uint32 {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}
