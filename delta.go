package rsync

import (
	"fmt"
	"io"

	"github.com/smtc/rollsum"
)

type delta struct {
	sig      *Signature
	pos      int
	weakSum  uint32
	blockLen uint32
	outer    io.Writer
	ms       matchStat
	mss      []matchStat
}

// dst.sig与src比较后，是否匹配的结果输出
type matchStat struct {
	match int   // 0: 未知状态，仅第一次出现,不能出现在最终结果中；1：匹配；-1：不匹配
	pos   int64 // 如果match为true，pos为dst文件的match位置，否则，pos为src中不匹配的起点位置
	// 当生成delta文件时，从src中读取该位置的数据，写入delta文件中
	length int64 // 如果match为true，length代表dst文件与src文件匹配的长度；否则，length代表src文件中
	// 没有匹配到的位置的总长度
}

const (
	// RS_OP_LITERAL_Nx与librsync中的定义不同
	// RS_OP_LITERAL_Nx = librsync RS_OP_LITERAL_Nx - 0x40
	// 这样，使用高位字节来存储数据压缩方式
	RS_OP_LITERAL_N1 uint8 = 0x01
	RS_OP_LITERAL_N2 uint8 = 0x02
	RS_OP_LITERAL_N4 uint8 = 0x03
	RS_OP_LITERAL_N8 uint8 = 0x04

	// 压缩方式
	RS_COMPRESS_NONE  uint8 = 0x00
	RS_COMPRESS_BZIP2 uint8 = 0x10
	RS_COMPRESS_GZIP  uint8 = 0x20
	RS_COMPRESS_LZW   uint8 = 0x30
	RS_COMPRESS_FLATE uint8 = 0x40

	RS_OP_COPY_N1_N1 uint8 = 0x45
	RS_OP_COPY_N1_N2 uint8 = 0x46
	RS_OP_COPY_N1_N4 uint8 = 0x47
	RS_OP_COPY_N1_N8 uint8 = 0x48
	RS_OP_COPY_N2_N1 uint8 = 0x49
	RS_OP_COPY_N2_N2 uint8 = 0x4a
	RS_OP_COPY_N2_N4 uint8 = 0x4b
	RS_OP_COPY_N2_N8 uint8 = 0x4c
	RS_OP_COPY_N4_N1 uint8 = 0x4d
	RS_OP_COPY_N4_N2 uint8 = 0x4e
	RS_OP_COPY_N4_N4 uint8 = 0x4f
	RS_OP_COPY_N4_N8 uint8 = 0x50
	RS_OP_COPY_N8_N1 uint8 = 0x51
	RS_OP_COPY_N8_N2 uint8 = 0x52
	RS_OP_COPY_N8_N4 uint8 = 0x53
	RS_OP_COPY_N8_N8 uint8 = 0x54
)

// generate delta
// param:
//     dstSig: reader of dst signature file
//     src: reader of src file
//     srcLen: src file content length
//     result: detla file writer
func GenDelta(dstSig io.Reader, src io.ReadSeeker, srcLen int64, result io.Writer) (err error) {
	var df delta

	// load signature file
	if df.sig, err = LoadSign(dstSig); err != nil {
		return
	}
	df.blockLen = df.sig.block_len
	df.outer = result

	if err = df.genDelta(src, srcLen); err != nil {
		return
	}

	err = df.flush(src)

	return
}

func (d *delta) genDelta(src io.ReadSeeker, srcLen int64) (err error) {
	var (
		c        byte
		p        []byte
		rs       rollsum.Rollsum
		rb       *rotateBuffer
		srcPos   int64
		matchAt  int64
		blockLen int
	)

	blockLen = int(d.sig.block_len)

	rb = NewRotateBuffer(srcLen, d.sig.block_len, src)
	p, srcPos, err = rb.rollFirst()
	if err == nil {
		// 计算初始weaksum
		rs.Init()
		rs.Update(p)
		for err == nil {
			// srcPos是当前读取src文件的绝对位置，matchAt对应于dstSig和dst文件的位置
			matchAt = d.findMatch(p, srcPos, rs.Digest())
			if matchAt < 0 {
				p, c, srcPos, err = rb.rollByte()
				if err != nil {
					break
				}
				rs.Rotate(c, p[blockLen-1])
			} else {
				p, srcPos, err = rb.rollBlock()
				rs.Init()
				if err != nil {
					break
				}
				rs.Update(p)
			}
		}
	} else if err == noBytesLeft {
		// reader没有内容
		println("reader has no content")
		err = nil
		return
	}

	if err != noBytesLeft && err != notEnoughBytes {
		// 出错
		return
	}

	if p, c, srcPos, err = rb.rollLeft(); err != nil {
		rs.Init()
		rs.Update(p)

		for err != nil {
			matchAt = d.findMatch(p, srcPos, rs.Digest())
			p, c, srcPos, err = rb.rollLeft()
			if err != nil {
				break
			}
			if matchAt > 0 {
				rs.Init()
				rs.Update(p)
			} else {
				rs.Rollout(c)
			}
		}
	}

	if err == noBytesLeft {
		println("last match stat:", d.ms.match, d.ms.pos, d.ms.length)
		d.mss = append(d.mss, d.ms)
		err = nil
	}

	return
}

// delta文件的header。
// 格式：
//    delta magic
func (d *delta) writeHeader() (err error) {
	_, err = d.outer.Write(htonl(DeltaMagic))
	return
}

// matchAt is basic file position
func (d *delta) findMatch(p []byte, pos int64, sum uint32) (matchAt int64) {
	matchAt = -1
	blocks, ok := d.sig.block_sigs[sum]
	if !ok {
		return
	}

	ssum := strongSum(p, d.sig.strong_sum_len)

	// 二分查找
	matchAt = blockSlice(blocks).search(ssum, pos, d.blockLen)
	if matchAt < 0 {
		if d.ms.match == 1 {
			// 上个匹配状态为匹配，重设ms
			d.mss = append(d.mss, d.ms)

			d.ms.match = -1
			d.ms.pos = pos
			d.ms.length = 1
		} else {
			// 上个匹配状态为不匹配，增加不匹配的长度
			if d.ms.match == 0 {
				d.ms.match = -1
			}
			d.ms.length++
		}
	} else {
		if d.ms.match == -1 {
			// 上个状态为不匹配, 重设ms
			d.mss = append(d.mss, d.ms)

			d.ms.match = 1
			d.ms.pos = matchAt
			d.ms.length = int64(len(p))
		} else {
			// 上个状态为初始状态或匹配状态
			if d.ms.match == 0 {
				d.ms.match = 1
			}
			d.ms.length += int64(len(p))
		}
	}

	return
}

// 打印matchstats
func (d *delta) dumpMatchStats(wr io.Writer) {
	pos := int64(0)

	for i, ms := range d.mss {
		switch ms.match {
		case 1:
			wr.Write([]byte(fmt.Sprintf("Match Block(%d): start at %d %d, length: %d\n", i, pos, ms.pos, ms.length)))
			pos += ms.length
		case -1:
			wr.Write([]byte(fmt.Sprintf("Miss  Block(%d): start at %d %d, length: %d\n", i, pos, ms.pos, ms.length)))
			pos += ms.length
		default:
			panic("ms.match should only be 1 or -1.")
		}
	}
}

// 根据matchStat写入delta文件
func (d *delta) flush(src io.ReadSeeker) (err error) {
	err = d.writeHeader()
	if err != nil {
		return
	}

	for _, ms := range d.mss {
		switch ms.match {
		case 1:
			if err = d.flushMatch(ms); err != nil {
				return
			}
		case -1:
			if err = d.flushMiss(ms, src); err != nil {
				return
			}
		default:
			panic("ms.match should only be 1 or -1.")
		}
	}
	// todo: delta文件结尾
	return nil
}

func int64Length(i uint64) uint8 {
	if i&uint64(0xffffffff00000000) != 0 {
		return 8
	} else if i&uint64(0xffffffffffff0000) != 0 {
		return 4
	} else if i&uint64(0xffffffffffffff00) != 0 {
		return 2
	}
	return 1
}

func intLength(i uint32) uint8 {
	if i&uint32(0xffff0000) != 0 {
		return 4
	} else if i&uint32(0xffffff00) != 0 {
		return 2
	}
	return 1
}

func (d *delta) flushMatch(ms matchStat) (err error) {
	var (
		cmd uint8
		buf []byte
	)

	whereBytes := int64Length(uint64(ms.pos))
	lenBytes := int64Length(uint64(ms.length))
	switch whereBytes {
	case 8:
		cmd = RS_OP_COPY_N8_N1
	case 4:
		cmd = RS_OP_COPY_N4_N1
	case 2:
		cmd = RS_OP_COPY_N2_N1
	case 1:
		cmd = RS_OP_COPY_N1_N1
	}
	switch lenBytes {
	case 8:
		cmd += 3
	case 4:
		cmd += 2
	case 2:
		cmd += 1
	case 1:
	}

	buf = append(buf, byte(cmd))
	buf = append(buf, vhtonll(uint64(ms.pos), int8(whereBytes))...)
	buf = append(buf, vhtonll(uint64(ms.length), int8(lenBytes))...)

	_, err = d.outer.Write(buf)
	return
}

func (d *delta) flushMiss(ms matchStat, src io.ReadSeeker) (err error) {
	var (
		n   int
		cmd uint8
		buf []byte
		tmp []byte
	)

	bytes := int64Length(uint64(ms.length))
	switch bytes {
	case 8:
		cmd = RS_OP_LITERAL_N8
	case 4:
		cmd = RS_OP_LITERAL_N4
	case 2:
		cmd = RS_OP_LITERAL_N2
	case 1:
		cmd = RS_OP_LITERAL_N1
	}
	buf = append(buf, byte(cmd))
	buf = append(buf, vhtonll(uint64(ms.length), int8(bytes))...)
	if _, err = d.outer.Write(buf); err != nil {
		return
	}
	if _, err = src.Seek(ms.pos, 0); err != nil {
		return
	}
	if ms.length <= 4096 {
		tmp = make([]byte, ms.length)
		if _, err = io.ReadFull(src, tmp); err != nil {
			return
		}
		if _, err = d.outer.Write(tmp); err != nil {
			return
		}
	} else {
		tmp = make([]byte, 4096)
		for n, err = io.ReadFull(src, tmp); err != nil; n, err = io.ReadFull(src, tmp) {
			if _, err = d.outer.Write(tmp); err != nil {
				return
			}
		}
		if err == io.ErrUnexpectedEOF {
			if _, err = d.outer.Write(tmp[0:n]); err != nil {
				return
			}
		} else if err == io.EOF {
			err = nil
		}
	}

	return
}
