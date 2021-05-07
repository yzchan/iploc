package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io/ioutil"
	"net/http"
)

type CopyWrite struct {
	ZipTag   [4]byte   // 固定值[43 5a 49 50]，即 CZIP
	DateCnt  uint32    // 1900.1.1到当前发布版本的天数
	_        [4]byte   // 未知 固定值 [01 00 00 00]
	FileSize uint32    // 文件大小
	_        [4]byte   // 未知数据
	Secret   uint32    // 密钥
	Version  [128]byte // 版本信息
	Link     [128]byte // 官网链接
}

func main() {
	fmt.Println("正在下载秘钥文件[http://update.cz88.net/ip/copywrite.rar]...")
	resp, err := http.Get("http://update.cz88.net/ip/copywrite.rar")
	if err != nil {
		panic(err)
	}
	fmt.Println("下载成功。开始读取秘钥...")
	defer resp.Body.Close()

	//buffer, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	panic(err)
	//}
	//secret := binary.LittleEndian.Uint32(buffer[20:24])

	cw := new(CopyWrite)
	binary.Read(resp.Body, binary.LittleEndian, cw)
	secret := cw.Secret
	//fmt.Println("读取秘钥成功！")

	//fmt.Println(secret)
	//version := string(buffer[24:142])
	version := string(cw.Version[:])
	enc := simplifiedchinese.GBK.NewDecoder()
	decoded, err := enc.String(version)

	fmt.Println(decoded)
	if err != nil {
		panic(err)
	}

	fmt.Println("正在下载加密数据文件[http://update.cz88.net/ip/qqwry.rar]...")
	resp, err = http.Get("http://update.cz88.net/ip/qqwry.rar")
	if err != nil {
		panic(err)
	}
	fmt.Println("下载成功。开始解码...")
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 512; i++ { // 处理前512字节
		secret = ((secret * 2053) + 1) & 0xFF    // 密钥变换
		data[i] = byte(uint32(data[i]) ^ secret) // 做异或运算解密对应字节
	}
	fmt.Println("开始解压文件...")
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	fmt.Println("开始保存qqwry.data数据文件...")
	qqwry, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("qqwry.dat", qqwry, 0777)
	if err != nil {
		panic(err)
	}
	fmt.Println("文件保存成功！")
}
