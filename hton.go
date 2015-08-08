package rsync

import (
	"encoding/binary"
	"io"
	"unsafe"
)

var (
	localOrder binary.ByteOrder
)

// Todo 2015-08-08:
// 使用binary/encoding的BigEndian/LittleEndian来重写

// host to network byte order transform

// 判断是BigEndian还是LittleEndian
// http://grokbase.com/t/gg/golang-nuts/129jhmdb3d/go-nuts-how-to-tell-endian-ness-of-machine
//
// x86，MOS Technology 6502，Z80，VAX，PDP-11等处理器为Little endian。
// Motorola 6800，Motorola 68000，PowerPC 970，System/370，SPARC（除V9外）等处理器为Big endian
// ARM, PowerPC (除PowerPC 970外), DEC Alpha, SPARC V9, MIPS, PA-RISC and IA64的字节序是可配置的。
// 网络传输一般采用大端序，也被称之为网络字节序，或网络序。IP协议中定义大端序为网络字节序。
func init() {
	var x uint32 = 0x01020304
	switch *(*byte)(unsafe.Pointer(&x)) {
	case 0x01:
		localOrder = binary.BigEndian

	case 0x04:
		localOrder = binary.LittleEndian
	}
}

func Htons(s uint16) []byte {
	var buf [2]byte

	binary.BigEndian.PutUint16(buf[0:2], s)
	return buf[0:2]
}

func Htonl(l uint32) []byte {
	var buf [4]byte

	binary.BigEndian.PutUint32(buf[0:4], l)
	return buf[0:4]
}

func Htonll(ll uint64) []byte {
	var buf [8]byte

	binary.BigEndian.PutUint64(buf[0:8], ll)
	return buf[0:8]
}

func htons(s int16) (res []byte) {
	res = make([]byte, 2)
	res[1] = byte(s & 0xff)
	res[0] = byte(s >> 8)

	return
}

func htonl(l uint32) (res []byte) {
	res = make([]byte, 4)
	res[0] = byte(l >> 24)
	res[1] = byte(l >> 16)
	res[2] = byte(l >> 8)
	res[3] = byte(l & 0xff)

	return
}

// 变长
func vhtonll(d uint64, bytes int8) []byte {
	buf := make([]byte, 8)
	for i := bytes - 1; i >= 0; i-- {
		buf[i] = uint8(d) /* truncated */
		d >>= 8
	}
	return buf[0:bytes]
}

// rd中保存的字节序为网络字序
// 根据参数l，从rd中读出l个字节，转换为uint64
// l最大为8
func vRead(rd io.Reader, l uint32) (i uint64, err error) {
	var buf [8]byte

	if l != 8 || l != 4 || l != 2 || l != 1 {
		panic("invalid param length")
	}
	_, err = io.ReadFull(rd, buf[0:l])
	if err != nil {
		return
	}
	switch l {
	case 1:
		i = uint64(buf[0])
	case 2:
		i = uint64(binary.BigEndian.Uint16(buf[0:2]))
	case 4:
		i = uint64(binary.BigEndian.Uint32(buf[0:4]))
	case 8:
		i = uint64(binary.BigEndian.Uint64(buf[0:8]))
	}

	return
}

// 从rd中读取8个字节 转换为uint64
func ntohll(rd io.Reader) (i uint64, err error) {
	var (
		n   int
		buf []byte = make([]byte, 8)
	)

	n, err = io.ReadFull(rd, buf)
	if n != 8 {
		return
	}

	i = uint64(uint64(buf[0])<<56 + uint64(buf[1])<<48 + uint64(buf[2])<<40 + uint64(buf[3])<<32 +
		uint64(buf[4])<<24 + uint64(buf[5])<<16 + uint64(buf[6])<<8 + uint64(buf[7]))

	return
}

// 从rd中读取4个字节 转换为uint32
func ntohl(rd io.Reader) (i uint32, err error) {
	var (
		n   int
		buf []byte = make([]byte, 4)
	)

	n, err = io.ReadFull(rd, buf)
	if n != 4 {
		return
	}

	i = uint32(uint32(buf[0])<<24 + uint32(buf[1])<<16 + uint32(buf[2])<<8 + uint32(buf[3]))

	return
}

// 从rd中读取2个字节，转换为int16
func ntohs(rd io.Reader) (i int16, err error) {
	var (
		n   int
		buf []byte = make([]byte, 2)
	)

	n, err = io.ReadFull(rd, buf)
	if n != 2 {
		return
	}

	i = int16(uint16(buf[0])<<8 + uint16(buf[1]))

	return
}

// 读出一个字节
func readByte(rd io.Reader) (i uint8, err error) {
	var buf [1]byte
	if _, err = rd.Read(buf[0:1]); err != nil {
		return
	}
	i = uint8(buf[0])
	return
}
