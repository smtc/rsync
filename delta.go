package rsync

import (
	"io"

	"github.com/smtc/rollsum"
)

type delta struct {
	sig      *Signature
	pos      int
	weakSum  uint32
	blockLen uint32
	outer    io.Writer
}

// generate delta
// param:
//     sigRd: reader of signature file
//     target: reader of target file
//     targetLen: target file content length
//     result: detla file writer
func GenDelta(sigRd, target io.Reader, targetLen int64, result io.Writer) (err error) {
	var (
		initial  bool
		c        byte
		p        []byte
		rs       rollsum.Rollsum
		rb       *rotateBuffer
		df       delta
		matchAt  int64
		blockLen int
	)

	// load signature file
	if df.sig, err = LoadSign(sigRd); err != nil {
		return
	}
	df.blockLen = df.sig.block_len
	df.outer = result
	err = df.writeHeader()
	if err != nil {
		return
	}

	blockLen = int(df.sig.block_len)

	rb = NewRotateBuffer(targetLen, blockLen, target)
	for p, c, initial, err = rb.rollByte(); err == nil; {
		if initial {
			// 计算初始weaksum
			rs.Init()
			rs.Update(p)
		}
		matchAt = df.findMatch(p, rb.rdLen-rb.left, rs.Digest())
		if matchAt < 0 {
			p, c, initial, err = rb.rollByte()
			if err != nil {
				break
			}
			rs.Rotate(c, p[len(p)-1])
		} else {
			rs.Init()
			p, err = rb.rollBlock()
			if err != nil {
				break
			}
			rs.Update(p)
		}
	}

	if err != noBytesLeft && err != notEnoughBytes {
		// 出错
		return
	} else {
		err = nil
	}

	if err == notEnoughBytes {
		//
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
	return
}
