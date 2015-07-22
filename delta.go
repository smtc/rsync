package rsync

import (
	"io"

	"github.com/smtc/rollsum"
)

type delta struct {
	sig     *Signature
	weakSum uint32
	pos     int
}

const defBufSize = 32768

// generate delta
// param:
//     sigRd: reader of signature file
//     target: reader of target file
//     targetLen: target file content length
//     result: detla file writer
func GenDelta(sigRd, target io.Reader, targetLen int64, result io.Writer) (err error) {
	var (
		n        int
		pos      int
		left     int64 // 当前文件的处理进度，还剩下多少字节未处理
		blockLen int
		buflen   int
		bufSize  int

		p, buf   []byte
		rs       rollsum.Rollsum
		df       delta
		matchAt  int
		matchErr error
	)

	// load signature file
	if df.sig, err = LoadSign(sigRd); err != nil {
		return
	}
	blockLen = int(df.sig.block_len)

	if int(targetLen) <= blockLen {
		// 文件大小不足blockLen
		return
	}

	left = targetLen
	bufSize = int(blockLen) << 2
	if bufSize < defBufSize {
		bufSize = defBufSize
	}
	buf = make([]byte, bufSize)
	n, err = io.ReadFull(target, buf[0:bufSize])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		// 出错
		return
	}
	// 计算初始weaksum
	rs.Init()
	rs.Update(buf)
	df.weakSum = rs.Digest()
	// 没有do {} while真是蛋疼！
	// 处理
	for {
		buflen = len(buf)
		pos = 0
		for int(pos+blockLen) <= buflen {
			p = buf[pos : pos+blockLen]

			matchAt, matchErr = df.findMatch(p)
			if matchAt == -1 {
				// 没有找到匹配项
				pos++
			} else {
				// 找到匹配项
				pos += blockLen
			}
		}

		if err != nil {
			// 读到文件结束
			break
		}
		// 继续读文件
		copy(buf[0:], buf[pos:])
		n, err = io.ReadFull(target, buf[pos:bufSize-pos])
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			// 出错
			return
		}

		buf = buf[0 : int(pos)+n]
	}

	// 正常结束，重置err
	err = nil
	// last block
	if left > 0 {
	}

	return
}

func (d *delta) findMatch(p []byte) (matchAt int, err error) {
	return
}
