package rsync

import (
	"errors"
	"io"
)

// 实现一个不断向前滚动的buffer

var (
	notEnoughBytes = errors.New("Not enough bytes")
)

type rotateBuffer struct {
	buffer   []byte
	blockLen int       // len(buffer)
	start    int       // point to buffer content start pos
	end      int       // point to buffer content end pos
	rd       io.Reader // reader to feed buffer
	eof      bool      // if reach reader eof
}

// total: rotateBuffer's buffer size
// blockLen: rotateBuffer block size
// rd: reader, which should feed the rotate buffer
func NewRotateBuffer(total, blockLen int, rd io.Reader) *rotateBuffer {
	var rb rotateBuffer

	rb.buffer = make([]byte, total)
	rb.blockLen = blockLen
	rb.rd = rd

	return &rb
}

func (rb *rotateBuffer) read() (n int, err error) {
	n, err = io.ReadFull(rb.rd, rb.buffer[rb.end:])
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		rb.eof = true
		err = nil
	}

	return
}

// 向前滚动1字节
// 分为以下几种情况：
//   1 buffer中尚未有数据：end=0，从reader中读取数据
//   2 buffer中数据不足一个blockLen时，将剩余的数据移到buffer的起始位置，从reader中读取数据
//   3
//
// 返回值：
//   p: 长度为blockLen的一段字节流
//   c：向前rotate时，roll out的字节，如果initial为true，该字节无效，因为没有roll out的字节
//   initial: 是否是第一次从reader中读取数据，此时，没有数据被roll out
//   err: 是否有错误, 遇到reader的结尾不会返回错误，要通过rotateBuffer的eof字段来判断是否结束
func (rb *rotateBuffer) rotate() (p []byte, c byte, initial bool, err error) {
	var (
		n int
	)

	if rb.end == 0 {
		// 1 第一次冲reader中读取数据
		initial = true
		n, err = rb.read()
		if err != nil {
			// reader出错
			return
		}
		if n < rb.blockLen {
			rb.buffer = rb.buffer[0:n]
			err = notEnoughBytes
			return
		}
		rb.end += rb.blockLen
		p = rb.buffer[0:rb.end]
		return
	}

	return
}

// rotateBuffer中最后一段不足blockLen的数据
func (rb *rotateBuffer) left() (p []byte) {
	return
}
