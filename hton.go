package rsync

import (
	"io"
)

// host to network byte order transform

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
func vhtonll(d uint64, bytes uint8) []byte {
	buf := make([]byte, 8)
	for i := bytes - 1; i >= 0; i-- {
		buf[i] = uint8(d) /* truncated */
		d >>= 8
	}
	return buf[0:bytes]
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
