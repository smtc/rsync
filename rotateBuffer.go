package rsync

import (
	"errors"
	"fmt"
	"io"
)

// 实现一个不断向前滚动的buffer
// rotateBuffer从reader中读入数据，放在缓存buffer中，每次rotate时，从buffer中消费blockLen，
// 当buffer中的数据消费完时，继续从reader中读取数据
//
// 有两种rotate的方式：一种是每次向前滚动一字节，另一种是每次向前滚动blockLen字节。
//
// 与rsync中处理方式不同，这里不对最后一个不足blockLen的字节流reduce操作，而是把最后一个不足blockLen
// 的字节流作为一个整体。

const (
	defBufSize  = 32768
	minBlockLen = 256
)

var (
	notEnoughBytes = errors.New("Not enough bytes") // 剩余数据不足一个blockLen
	noBytesLeft    = errors.New("no bytes left")    // EOF, 没有数据
)

type rotateBuffer struct {
	buffer   []byte
	rdLen    int64     // the reader file total length
	blockLen int       // block length
	bufSize  int       // buffer size
	left     int64     // how many bytes not read
	start    int       // point to buffer content start pos
	end      int       // point to buffer content end pos
	rd       io.Reader // reader to feed buffer
	eof      bool      // if reach reader eof
}

// total: rotateBuffer's buffer size
// blockLen: rotateBuffer block size
// rd: reader, which should feed the rotate buffer
func NewRotateBuffer(total int64, blockLen int, rd io.Reader) *rotateBuffer {
	var rb rotateBuffer

	rb.rdLen = total
	rb.bufSize = blockLen << 4
	if rb.bufSize < defBufSize {
		rb.bufSize = defBufSize
	}
	rb.buffer = make([]byte, rb.bufSize)
	rb.blockLen = blockLen
	rb.rd = rd

	return &rb
}

// 触发read只有两种可能：
//   1 初始状态，还没有从reader中读过任何数据，此时end=0
//   2 内部buffer已经消费到结尾，此时end>=bufSize
func (rb *rotateBuffer) read() (n int, err error) {
	if rb.end == 0 {
		n, err = io.ReadFull(rb.rd, rb.buffer[0:])
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			// reader的数据不足bufSize
			rb.eof = true
			err = nil
			rb.buffer = rb.buffer[0:n]
		}
		rb.left = rb.rdLen
		rb.end = len(rb.buffer)
		if rb.end > rb.blockLen {
			rb.end = rb.blockLen
		}
		return
	}

	if rb.eof {
		err = noBytesLeft
		return
	}
	// 第2种情况
	// 首先，将buffer中未消费的数据移到buffer的头部
	// 然后，从reader中读入数据，追加再buffer中
	// 重新设置rb的start, end
	copy(rb.buffer[0:], rb.buffer[rb.start:])
	rb.end -= rb.start
	n, err = io.ReadFull(rb.rd, rb.buffer[rb.end:])
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		rb.eof = true
		err = nil
		rb.buffer = rb.buffer[0 : rb.end+n]
	}
	rb.start = 0
	rb.end = len(rb.buffer)
	if rb.end > rb.blockLen {
		rb.end = rb.blockLen
	}

	return
}

// 向前滚动1字节
// 分为以下几种情况：
//   1 buffer中尚未有数据：end=0，从reader中读取数据
//   2 buffer中未使用的数据不足一个blockLen时，将剩余的数据移到buffer的起始位置，从reader中读取数据
//   3
//
// 返回值：
//   p: 长度为blockLen的一段字节流
//   c：向前rotate时，roll out的字节，如果initial为true，该字节无效，因为没有roll out的字节
//   initial: 是否是第一次从reader中读取数据，此时，没有数据被roll out
//   err: 是否有错误, 遇到reader的结尾不会返回错误，要通过rotateBuffer的eof字段来判断是否结束
func (rb *rotateBuffer) rollByte() (p []byte, c byte, initial bool, err error) {
	var (
		n int
	)

	if rb.end == 0 {
		// 1 第一次冲reader中读取数据
		initial = true
		n, err = rb.read()
		if err != nil { // reader出错
			return
		}
		if n < rb.blockLen {
			err = notEnoughBytes
			return
		}
		rb.left -= int64(rb.blockLen)
		p = rb.buffer[0:rb.end]
		return
	}

	if rb.end == rb.bufSize {
		if rb.eof {
			err = noBytesLeft
			return
		}
		// 继续从reader中读入数据
		c = rb.buffer[rb.start]
		rb.start++
		n, err = rb.read() // rb.start在rb.read()中被置为0
		if err != nil {
			return
		}
		if rb.end < rb.blockLen {
			err = notEnoughBytes
			return
		}
		rb.left--
		p = rb.buffer[rb.start:rb.end]
		return
	}
	c = rb.buffer[rb.start]
	rb.start++
	rb.end++
	rb.left--
	p = rb.buffer[rb.start:rb.end]
	return
}

func Assert(c bool, msg string) {
	if !c {
		panic(msg)
	}
}

func Assertf(c bool, format string, args ...interface{}) {
	if !c {
		panic(fmt.Sprintf(format, args))
	}
}

func (rb *rotateBuffer) rollBlock() (p []byte, err error) {
	if rb.left == 0 {
		err = noBytesLeft
		return
	}

	if rb.left < int64(rb.blockLen) {
		if !rb.eof {
			rb.start = rb.end
			Assertf(len(rb.buffer) == rb.bufSize,
				"left(%d)<blockLen(%d), eof=false, rotate buffer length %d should equal with bufSize %d\n",
				rb.left, rb.blockLen, len(rb.buffer), rb.bufSize)
			rb.end = rb.bufSize

			_, err = rb.read()
			if err != nil {
				return
			}
		}
		// 剩余未读已不足一个blockLen
		err = notEnoughBytes
		return
	}

	if rb.end+rb.blockLen > len(rb.buffer) {
		if rb.eof {
			err = notEnoughBytes
			return
		}
		Assertf(len(rb.buffer) == rb.bufSize,
			"start=%d, end=%d, eof=flase, rotate buffer length %d should equal with bufSize %d\n",
			rb.start, rb.end, len(rb.buffer), rb.bufSize)

		rb.start += rb.blockLen
		rb.end = len(rb.buffer)
		_, err = rb.read()
		if err != nil {
			return
		}
		p = rb.buffer[rb.start:rb.end]
		rb.left -= int64(rb.blockLen)
		return
	}

	rb.start += rb.blockLen
	rb.end += rb.blockLen
	rb.left -= int64(rb.blockLen)
	p = rb.buffer[rb.start:rb.end]

	return
}

// rotateBuffer中最后一段不足blockLen的数据
func (rb *rotateBuffer) rollLeft() (p []byte) {
	p = rb.buffer[rb.end:]
	rb.end++
	return
}
