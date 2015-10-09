package rsync

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"

	"github.com/smtc/seekbuffer"
)

// 将输入的data随机分成两部分，一部分作为源，一部分作为目标，最终经过几个步骤后，目标与源相同
// 1 随机分为两部分， src, dst
// 2 对dst做signature， dst.sig
// 3 根据dst.sig和src做delta, dst-src.delta
// 4 使用dst-src.delta patch dst,得到新的target
// 5 比较target与src
func Fuzz(data []byte) int {
	n := len(data)
	if n < 2 {
		return -1
	}
	pos := randn(n)
	src := data[0:pos]
	dst := data[pos:]
	blocklens := []int{2, 4, 8, 16}
	for _, b := range blocklens {
		ret := doFuzz(src, dst, b, false)
		if ret == 0 {
			return 0
		}
	}

	return 1
}

func randn(n int) int {
	ret := 0
	for ret != 0 {
		ret = rand.Intn(n)
	}

	return ret
}

// if success, return 1
// if failed, return 0
func doFuzz(src, dst []byte, b int, debug bool) int {
	var (
		err    error
		dstRd  *bytes.Buffer
		srcSr  *seekbuffer.SeekBuffer
		dstSr  *seekbuffer.SeekBuffer
		sign   = new(bytes.Buffer)
		delta  = new(bytes.Buffer)
		target = new(bytes.Buffer)
	)

	dstRd = bytes.NewBuffer(dst)

	err = GenSign(dstRd, int64(len(dst)), uint32(b), sign)
	if err != nil {
		if debug {
			log.Printf("GenSign failed: %s\n", err.Error())
		}
		return 0
	}

	srcSr = seekbuffer.NewSeekBuffer(src)
	err = GenDelta(sign, srcSr, int64(len(src)), delta)
	if err != nil {
		if debug {
			log.Printf("GenDelta failed: %s\n", err.Error())
		}
		return 0
	}

	dstSr = seekbuffer.NewSeekBuffer(dst)

	err = Patch(delta, dstSr, target)
	if err != nil {
		if debug {
			log.Printf("Patch failed: %s\n", err.Error())
		}
		return 0
	}

	tbuf := target.Bytes()
	if bytes.Compare(tbuf, src) == 0 {
		return 1
	}
	if debug {
		log.Printf("target NOT equal with src: src=[%s] dst=[%s] target=[%s]\n",
			hexdump(src), hexdump(dst), hexdump(tbuf))
	}
	return 0
}

func hexdump(buf []byte) string {
	ret := ""
	for _, b := range buf {
		ret += fmt.Sprintf("%02x ", b)
	}
	return ret
}
