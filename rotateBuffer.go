package rsync

import (
	"errors"
	"fmt"
	"io"
	"log"
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
	absHead  int64     // 当前block的开始处在文件中的绝对位置, 0 index
	absTail  int64     // 当前block的结束处在文件中的绝对位置, 0 index
	start    int       // point to buffer content start pos
	end      int       // point to buffer content end pos
	rd       io.Reader // reader to feed buffer
	eof      bool      // if reach reader eof
}

// total: rotateBuffer's buffer size
// blockLen: rotateBuffer block size
// rd: reader, which should feed the rotate buffer
func NewRotateBuffer(total int64, blockLen uint32, rd io.Reader) *rotateBuffer {
	var rb rotateBuffer

	rb.rdLen = total
	//rb.bufSize = blockLen << 4
	rb.bufSize = int(blockLen * 2)
	if rb.bufSize < defBufSize {
		//	rb.bufSize = defBufSize
	}
	rb.buffer = make([]byte, rb.bufSize)
	rb.blockLen = int(blockLen)
	rb.rd = rd

	return &rb
}

// 触发read只有两种可能：
//   1 初始状态，还没有从reader中读过任何数据，此时end=0
//   2 内部buffer已经消费到结尾，此时end>=bufSize
func (rb *rotateBuffer) read() (n int, err error) {
	Assertf(rb.absTail != 0, "this call should NOT be first read, absTail should NOT be 0")

	if rb.eof {
		log.Printf("rotate buffer is EOF, should NOT call read.\n")
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
	if err == io.EOF {
		return n, noBytesLeft
	} else if err == io.ErrUnexpectedEOF {
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

// 第一次读
func (rb *rotateBuffer) rollFirst() (p []byte, pos int64, err error) {
	Assertf(rb.absTail == 0, "first read, absTail should be 0")
	var n int

	// 1 第一次冲reader中读取数据
	n, err = io.ReadFull(rb.rd, rb.buffer[0:])
	switch err {
	case io.EOF:
		err = noBytesLeft
		return
	case io.ErrUnexpectedEOF:
		err = nil
		rb.eof = true
		rb.buffer = rb.buffer[0:n]
		if n < rb.blockLen {
			err = notEnoughBytes
			rb.end = n
			rb.absTail = int64(n)
			p = rb.buffer
			return
		}
		fallthrough
	case nil:
		rb.absTail = int64(n)
		rb.end = n
		if n > rb.blockLen {
			rb.absTail = int64(rb.blockLen)
			rb.end = rb.blockLen
		}

	default:
		return
	}

	p = rb.buffer[0:rb.end]
	return
}

// 向前滚动1字节
// 分为以下几种情况：
//   2 buffer中未使用的数据不足一个blockLen时，将剩余的数据移到buffer的起始位置，从reader中读取数据
//   3
//
// 返回值：
//   p: 长度为blockLen的一段字节流
//   c：向前rotate时，roll out的字节，如果initial为true，该字节无效，因为没有roll out的字节
//   pos: 读取的字节在rd中的绝对位置，例如该字节在文件中的位置
//   err: 是否有错误, 遇到reader的结尾不会返回错误，要通过rotateBuffer的eof字段来判断是否结束
func (rb *rotateBuffer) rollByte() (p []byte, c byte, pos int64, err error) {
	rb.start++
	rb.absHead++
	if rb.absTail >= rb.rdLen {
		err = notEnoughBytes
		return
	}

	if rb.end == rb.bufSize {
		// 继续从reader中读入数据
		c = rb.buffer[rb.start-1]
		_, err = rb.read() // rb.start在rb.read()中被置为0
		if err != nil {
			return
		}

		rb.absTail++
		p = rb.buffer[rb.start:rb.end]
		pos = rb.absHead
		return
	}
	c = rb.buffer[rb.start-1]
	rb.end++
	rb.absTail++
	p = rb.buffer[rb.start:rb.end]
	pos = rb.absHead
	return
}

func Assert(c bool, msg string) {
	if !c {
		panic(msg)
	}
}

func Assertf(c bool, format string, args ...interface{}) {
	if !c {
		panic(fmt.Sprintf(format, args...))
	}
}

func (rb *rotateBuffer) rollBlock() (p []byte, pos int64, err error) {
	rb.start += rb.blockLen
	rb.absHead += int64(rb.blockLen)
	if rb.absTail+int64(rb.blockLen) > rb.rdLen {
		// 剩余未读已不足一个blockLen
		rb.absTail = rb.rdLen
		err = notEnoughBytes
		return
	}

	if rb.end+rb.blockLen > len(rb.buffer) {
		/*
			Assertf(rb.eof == false, "Here, eof should be false")
			Assertf(len(rb.buffer) == rb.bufSize,
				"start=%d, end=%d, eof=flase, rotate buffer length %d should equal with bufSize %d\n",
				rb.start, rb.end, len(rb.buffer), rb.bufSize)
		*/
		rb.end = len(rb.buffer)
		_, err = rb.read()
		if err != nil {
			return
		}
		rb.absTail += int64(rb.blockLen)
		p = rb.buffer[rb.start:rb.end]
		pos = rb.absHead

		return
	}

	rb.end += rb.blockLen
	rb.absTail += int64(rb.blockLen)
	p = rb.buffer[rb.start:rb.end]
	pos = rb.absHead

	return
}

// 此时TailHead应该已经是rdLen
// rotateBuffer中最后一段不足blockLen的数据
func (rb *rotateBuffer) rollLeft() (p []byte, c byte, pos int64, err error) {
	if !rb.eof {
		//fmt.Println("rollLeft: not eof, read all")
		rb.start = rb.end
		/*
			Assertf(len(rb.buffer) == rb.bufSize,
				"tail(%d)+blockLen(%d)>rdLen(%), eof=false, rotate buffer length
				 %d should equal with bufSize %d\n",
				rb.absTail, rb.blockLen, rb.rdLen, len(rb.buffer), rb.bufSize)
		*/
		rb.end = rb.bufSize

		_, err = rb.read()
		if err == noBytesLeft {
			err = nil
		} else if err != nil {
			return
		}
	}

	if rb.absHead >= rb.rdLen {
		//fmt.Println("end:", rb.absHead, rb.rdLen)
		err = noBytesLeft
		return
	}
	if rb.start > 0 {
		c = rb.buffer[rb.start-1]
	}
	p = rb.buffer[rb.start:]
	pos = rb.absHead
	rb.start++
	rb.absHead++
	return
}
