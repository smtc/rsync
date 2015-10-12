package rsync

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	//"log"

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
	debug    bool
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
func GenDelta(dstSig io.Reader,
	src io.ReadSeeker,
	srcLen int64,
	result io.Writer,
	args ...bool) (err error) {
	var (
		df delta
	)

	if len(args) > 0 {
		df.debug = args[0]
	}
	// load signature file
	if df.sig, err = LoadSign(dstSig, df.debug); err != nil {
		err = errors.New("Load Signature failed: " + err.Error())
		return
	}
	df.blockLen = df.sig.block_len
	df.outer = result

	if err = df.genDelta(src, srcLen); err != nil {
		err = errors.New("generate Delta failed: " + err.Error())
		return
	}

	// 打印调试信息
	if df.debug {
		df.dump()
	}

	if err = df.flush(src); err != nil {
		err = errors.New("write Delta failed: " + err.Error())
	}

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

	if d.debug {
		d.dumpSign()
	}

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
		if d.debug {
			fmt.Println("reader has no content:", srcLen)
		}
		err = nil
		return
	}

	if err != noBytesLeft && err != notEnoughBytes {
		// 出错
		return
	}

	if d.debug {
		fmt.Printf("rotate buffer left no more than a block: block=%d start=%d end=%d absHead=%d absTail=%d eof=%v\n",
			blockLen, rb.start, rb.end, rb.absHead, rb.absTail, rb.eof)
	}

	if p, c, srcPos, err = rb.rollLeft(); err == nil {
		rs.Init()
		rs.Update(p)

		for err == nil {
			matchAt = d.findMatch(p, srcPos, rs.Digest())

			if matchAt >= 0 {
				// 剩余的内容已经匹配到，不需要继续处理
				break
			} else {
				p, c, srcPos, err = rb.rollLeft()
				if err != nil {
					break
				}
				rs.Rollout(c)
			}
		}
	} else {
		if d.debug {
			fmt.Println(string(p), c, srcPos, err)
		}
	}

	if err == noBytesLeft || err == nil {
		if d.debug {
			fmt.Println("last match stat:", d.ms.match, d.ms.pos, d.ms.length)
		}
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
	if blocks, ok := d.sig.block_sigs[sum]; ok {
		ssum := strongSum(p, d.sig.strong_sum_len)
		// 二分查找
		matchAt = blockSlice(blocks).search(ssum, pos, d.blockLen)
	}

	if matchAt < 0 {
		if d.ms.match == 1 {
			// 上个匹配状态为匹配，重设ms
			d.mss = append(d.mss, d.ms)
			if d.debug {
				fmt.Printf("  delta match: pos=%d len=%d\n", d.ms.pos, d.ms.length)
			}

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
		// 找到匹配
		if d.ms.match == -1 {
			// 上个状态为不匹配, 重设ms
			d.mss = append(d.mss, d.ms)
			if d.debug {
				fmt.Printf("  delta miss: pos=%d len=%d\n", d.ms.pos, d.ms.length)
			}

			d.ms.match = 1
			d.ms.pos = matchAt
			d.ms.length = int64(len(p))
		} else {
			// 上个状态为初始状态或匹配状态
			if d.ms.match == 0 {
				d.ms.match = 1
				d.ms.pos = matchAt
				d.ms.length = int64(len(p))
			} else {
				// 2015-08-09
				// 检查与上一个匹配是否能够合并
				if d.ms.pos+d.ms.length == matchAt {
					d.ms.length += int64(len(p))
				} else {
					fmt.Printf("   !!! This match Cannot merge with the last match!!!\n")
					d.mss = append(d.mss, d.ms)
					if d.debug {
						fmt.Printf("  delta match (not merged!!!): pos=%d len=%d\n",
							d.ms.pos, d.ms.length)
					}

					d.ms.match = 1
					d.ms.pos = matchAt
					d.ms.length = int64(len(p))
				}
			}
		}
	}

	return
}

func (d *delta) dumpSign() {
	var s string
	sig := d.sig
	s = fmt.Sprintf("dump Signature:\n src length: %d blocks: %d tlen: %d block len: %d sum len: %d magic: 0x%x\n",
		sig.flength, sig.count, sig.flength, sig.block_len, sig.strong_sum_len, sig.magic)
	for _, block_sigs := range sig.block_sigs {
		//s += fmt.Sprintf(" block sum: 0x%x:\n", sum)
		for _, block_sig := range block_sigs {
			s += fmt.Sprintf("    block index: %d block weak sum: 0x%x strong sum: %x\n", block_sig.i, block_sig.wsum, block_sig.ssum)
		}
	}
	fmt.Println(s)
}

// 比较两个MatchStats是否相同
func (d1 *delta) equalMatchStats(d2 *delta) bool {
	if len(d1.mss) != len(d2.mss) {
		return false
	}

	for i := 0; i < len(d1.mss); i++ {
		m1 := d1.mss[i]
		m2 := d2.mss[i]
		if m1.length != m2.length || m1.match != m2.match || m1.pos != m2.pos {
			return false
		}
	}
	return true
}

// 打印到终端，调试用
func (d *delta) dump() {
	buf := &bytes.Buffer{}
	d.dumpMatchStats(buf)
	fmt.Println(string(buf.Bytes()))
}

// 打印matchstats
func (d *delta) dumpMatchStats(wr io.Writer) {
	pos := int64(0)

	wr.Write([]byte(fmt.Sprintf("\nDelta info: blockLen=%d matchBlock=%d\n",
		d.blockLen, len(d.mss))))
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

// 生成delta文件
// 1 delta文件头
// 2 将matchStat写入delta文件
//
func (d *delta) flush(src io.ReadSeeker) (err error) {
	err = d.writeHeader()
	if err != nil {
		return
	}

	for _, ms := range d.mss {
		switch ms.match {
		case 1:
			if err = d.flushMatch(ms); err != nil {
				panic(fmt.Sprintf("flushMatch failed: %s matchStat: %v", err.Error(), ms))
				return
			}
		case -1:
			if err = d.flushMiss(ms, src); err != nil {
				panic(fmt.Sprintf("flushMiss failed: %s: matchStat: %v", err.Error(), ms))
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

// cmd:    1字节
// pos:    变长，1,2,4,8字节，根据cmd决定
// length: 变长：1,2,4,8字节，根据cmd决定
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

	if d.debug {
		fmt.Printf("   flush Match [where=%d len=%d], buf length: %d\n",
			ms.pos, ms.length, len(buf))
	}
	return
}

// cmd:    1字节
// length: 变长：1,2,4,8字节，根据cmd决定
// 内容区:  变长，长度=length
// 2015-08-10: todo: 数据压缩
func (d *delta) flushMiss(ms matchStat, src io.ReadSeeker) (err error) {
	var (
		//n   int
		ml  int64 // miss length
		cmd uint8
		hdr []byte
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

	// 写入miss block头部
	hdr = append(hdr, byte(cmd))
	hdr = append(hdr, vhtonll(uint64(ms.length), int8(bytes))...)
	if _, err = d.outer.Write(hdr); err != nil {
		return
	}
	if _, err = src.Seek(ms.pos, 0); err != nil {
		err = errors.New("Seek failed: " + err.Error())
		return
	}

	ml = ms.length
	buf = make([]byte, 4096)
	for err == nil && ml > 0 {
		if ml >= 4096 {
			tmp = buf[0:4096]
			ml -= 4096
		} else {
			tmp = buf[0:ml]
			ml = 0
		}
		// 不应该出现错误
		_, err = src.Read(tmp)
		if err != nil {
			panic(fmt.Sprintf("pipe: read failed: expect %d, error: %s", len(tmp), err.Error()))
		}
		if _, err = d.outer.Write(tmp); err != nil {
			return
		}
	}
	/*
		if ms.length <= 4096 {
			tmp = make([]byte, ms.length)
			if _, err = io.ReadFull(src, tmp); err != nil {
				return
			}
			if _, err = d.outer.Write(tmp); err != nil {
				return
			}
		} else {
			ml = d.ms.length
			tmp = make([]byte, 4096)
			n, err = io.ReadFull(src, tmp)
			for err == nil && ml > 0 {
				if _, ierr := d.outer.Write(tmp); ierr != nil {
					err = fmt.Errorf("flushMiss failed: write output faile: %s", err.Error())
					return
				}
				n, err = io.ReadFull(src, tmp)
			}
			if err == io.ErrUnexpectedEOF {
				if _, err = d.outer.Write(tmp[0:n]); err != nil {
					return
				}
			} else if err == io.EOF {
				err = nil
			}
		}
	*/
	if d.debug {
		fmt.Printf("   flush miss [where=%d len=%d], hdr len: %d miss len: %d\n",
			ms.pos, ms.length, len(hdr), ms.length)
	}
	return
}
