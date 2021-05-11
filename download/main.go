package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"
)

//type CopyWrite struct {
//	ZipTag   [4]byte   // 固定值[43 5a 49 50]，即 CZIP
//	DateCnt  uint32    // 1900.1.1到当前发布版本的天数
//	_        [4]byte   // 未知 固定值 [01 00 00 00]
//	FileSize uint32    // 文件大小
//	_        [4]byte   // 未知数据
//	Secret   uint32    // 密钥
//	Version  [128]byte // 版本信息
//	Link     [128]byte // 官网链接
//}

func download(uri string) (stream []byte, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return
	}
	filename := filepath.Base(u.Path)

	client := http.DefaultClient
	client.Timeout = time.Second * 60 //设置超时时间
	resp, err := client.Get(uri)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("[%s]下载失败,%s.", filename, resp.Status))
		return
	}

	log.Printf("[INFO] 正在下载: [%s]", filename)

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return
	}

	reader := io.LimitReader(resp.Body, int64(size))
	writer := new(bytes.Buffer)
	// start new bar
	bar := pb.Full.Start64(int64(size))

	// create proxy reader
	barReader := bar.NewProxyReader(reader)
	// copy from proxy reader
	n, err := io.Copy(writer, barReader)
	if err != nil || n != int64(size) {
		return
	}
	// finish bar
	bar.Finish()
	stream = writer.Bytes()
	return
}

func main() {
	log.Println("开始下载密钥文件")
	keyStream, err := download("http://update.cz88.net/ip/copywrite.rar")
	if err != nil {
		panic(err)
	}

	secret := binary.LittleEndian.Uint32(keyStream[20:24])

	version := string(keyStream[24:142])
	enc := simplifiedchinese.GBK.NewDecoder()
	decoded, err := enc.String(version)

	log.Println(decoded)

	log.Println("开始下载数据文件")
	dataStream, err := download("http://update.cz88.net/ip/qqwry.rar")
	if err != nil {
		panic(err)
	}

	log.Println("开始解密文件")
	for i := 0; i < 512; i++ { // 处理前512字节
		secret = ((secret * 2053) + 1) & 0xFF                // 密钥变换
		dataStream[i] = byte(uint32(dataStream[i]) ^ secret) // 做异或运算解密对应字节
	}

	reader, err := zlib.NewReader(bytes.NewReader(dataStream))
	if err != nil {
		panic(err)
	}
	log.Println("开始保存qqwry.data数据文件...")
	qqwry, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("qqwry.dat", qqwry, 0777)
	if err != nil {
		panic(err)
	}
	log.Println("文件保存成功！")
}
