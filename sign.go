package rsync

import (
	"bytes"
	"errors"
	"io"
	"sort"
)

var (
	defaultBlockLen uint32 = 2048
	defaultSumLen   uint32 = 64
)

type SignHdr struct {
	magic    uint32
	blockLen uint32
	sumLen   uint32
}

func (hdr *SignHdr) toBytes() (res []byte) {
	res = append(res, htonl(hdr.magic)...)
	res = append(res, htonl(hdr.blockLen)...)
	res = append(res, htonl(hdr.sumLen)...)
	return
}

// signature header:
//   magic    4 bytes
func signHeader(sumLen, blockLen uint32) (hdr SignHdr) {
	hdr.magic = BlakeMagic

	hdr.blockLen = blockLen
	hdr.sumLen = sumLen

	return
}

// generates signature
func GenSign(rd io.Reader, sumLen, blockLen uint32, result io.Writer) (err error) {
	var (
		n   int
		hdr SignHdr
		buf []byte
		sig []byte
	)

	if sumLen == 0 {
		sumLen = defaultSumLen
	}
	if blockLen == 0 {
		blockLen = defaultBlockLen
	}

	hdr = signHeader(sumLen, blockLen)
	sig = append(sig, hdr.toBytes()...)
	result.Write(sig)

	buf = make([]byte, blockLen)

	for {
		/*
			ReadFull reads exactly len(buf) bytes from r into buf. It returns
			the number of bytes copied and an error if fewer bytes were read.
			The error is EOF only if no bytes were read. If an EOF happens after
			reading some but not all the bytes, ReadFull returns ErrUnexpectedEOF.
			On return, n == len(buf) if and only if err == nil.
		*/
		n, err = io.ReadFull(rd, buf)
		if uint32(n) == blockLen || err == io.ErrUnexpectedEOF {
			wsum := weakSum(buf)
			result.Write(htonl(wsum))

			ssum := strongSum(buf, sumLen)
			result.Write(ssum)
		}
		if err != nil {
			break
		}
	}

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}

	return
}

func LoadSign(rd io.Reader) (sig *Signature, err error) {
	var (
		count int
		block rs_block_sig
	)

	sig = new(Signature)

	if sig.magic, err = ntohl(rd); err != nil {
		return
	}
	if sig.block_len, err = ntohl(rd); err != nil {
		return
	}
	if sig.strong_sum_len, err = ntohl(rd); err != nil {
		return
	}
	block.ssum = make([]byte, sig.strong_sum_len)
	// read weak sum & strong sum
	for {
		block.i = count
		if block.wsum, err = ntohl(rd); err != nil {
			return
		}

		if _, err = io.ReadFull(rd, block.ssum); err != nil && err != io.EOF {
			return
		}

		sig.block_sigs = append(sig.block_sigs, block)
		count++
	}
	if count == 0 {
		err = errors.New("No signature block found.")
		return
	}
	sig.count = count

	// build
	err = buildSign(sig)

	return
}

func gettag(sum uint32) uint16 {
	var (
		a, b uint16
	)

	a = uint16(sum & 0xffff)
	b = uint16(sum >> 16)
	return uint16((a + b) & 0xffff)
}

func buildSign(sig *Signature) (err error) {
	i := 0

	sig.tag_table = make([]rs_tag_table_entry, TABLE_SIZE)
	sig.targets = make([]rs_target, sig.count)

	for i = 0; i < sig.count; i++ {
		sig.targets[i].i = i
		sig.targets[i].t = gettag(sig.block_sigs[i].wsum)
	}

	// 对sig.targets排序, 按照target.t从小到大排列
	sort.Sort(sig)

	// initialize
	for i = 0; i < TABLE_SIZE; i++ {
		sig.tag_table[i].l = NULL_TAG
		sig.tag_table[i].r = NULL_TAG
	}

	for i = sig.count - 1; i >= 0; i++ {
		sig.tag_table[sig.targets[i].t].l = i
	}
	for i = 0; i < sig.count; i++ {
		sig.tag_table[sig.targets[i].t].r = i
	}

	return
}

// sort rs_target interface

func (s *Signature) Len() int {
	return len(s.targets)
}

func (s *Signature) Swap(i, j int) {
	s.targets[i], s.targets[j] = s.targets[j], s.targets[i]
}

func (s *Signature) Less(i, j int) bool {
	ti := s.targets[i]
	tj := s.targets[j]
	// first, use target t
	if ti.t > tj.t {
		return false
	} else if ti.t < tj.t {
		return true
	}

	wi := s.block_sigs[ti.i].wsum
	wj := s.block_sigs[tj.i].wsum
	// then, use the weaksum to compare
	if wi > wj {
		return false
	} else if wi < wj {
		return true
	}

	// finally, use the strong sum to compare

	// Compare returns an integer comparing two byte slices lexicographically.
	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
	// A nil argument is equivalent to an empty slice.
	c := bytes.Compare(s.block_sigs[ti.i].ssum, s.block_sigs[tj.i].ssum)
	if c < 0 {
		return true
	}
	return false
}
