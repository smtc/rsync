package rsync

import (
	"bytes"
	"fmt"
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
	totalLen int64 //总长度
}

func (hdr *SignHdr) toBytes() (res []byte) {
	res = append(res, htonl(hdr.magic)...)
	res = append(res, htonl(hdr.blockLen)...)
	res = append(res, htonl(hdr.sumLen)...)
	res = append(res, vhtonll(uint64(hdr.totalLen), 8)...)
	return
}

// signature header:
//   magic    4 bytes
func signHeader(rdLen int64, sumLen, blockLen uint32) (hdr SignHdr) {
	hdr.magic = BlakeMagic

	hdr.blockLen = blockLen
	hdr.sumLen = sumLen
	hdr.totalLen = rdLen

	return
}

// generates signature
func GenSign(rd io.Reader, rdLen int64, blockLen uint32, result io.Writer) (err error) {
	var (
		n      int
		sumLen uint32 = 64
		hdr    SignHdr
		buf    []byte
		sig    []byte
	)

	if blockLen == 0 {
		blockLen = defaultBlockLen
	}

	hdr = signHeader(rdLen, sumLen, blockLen)
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
			wsum := weakSum(buf[0:n])
			result.Write(htonl(wsum))

			ssum := strongSum(buf[0:n], sumLen)
			//fmt.Printf("Sign: length=%d p=%s wsum=0x%x ssum=0x%x\n", n, string(buf[0:n]), wsum, string(ssum))
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

func LoadSign(rd io.Reader, debug bool) (sig *Signature, err error) {
	var (
		n      int
		ok     bool
		count  int
		tlen   uint64
		block  *rs_block_sig
		blocks []*rs_block_sig
		tag    *rs_tag_table_entry
	)

	sig = new(Signature)

	if sig.magic, err = ntohl(rd); err != nil {
		err = fmt.Errorf("read signature maigin failed: %s", err.Error())
		return
	}
	if sig.block_len, err = ntohl(rd); err != nil {
		err = fmt.Errorf("read signature block lenght failed: %s", err.Error())
		return
	}
	if sig.strong_sum_len, err = ntohl(rd); err != nil {
		err = fmt.Errorf("read signature strong sum length failed: %s", err.Error())
		return
	}
	if tlen, err = ntohll(rd); err != nil {
		err = fmt.Errorf("read signature remainer length failed: %s", err.Error())
		return
	}
	sig.flength = int64(tlen)

	// 初始化map
	sig.block_sigs = make(map[uint32][]*rs_block_sig)
	sig.tag_tables = make(map[uint32]*rs_tag_table_entry)

	if tlen == 0 {
		return
	}

	// read weak sum & strong sum
	for {
		block = new(rs_block_sig)
		block.ssum = make([]byte, sig.strong_sum_len)
		block.i = count
		if block.wsum, err = ntohl(rd); err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}

		if n, err = io.ReadFull(rd, block.ssum); err != nil {
			err = fmt.Errorf("LoadSign: read strong sum failed: n=%d expect=%d error=%s",
				n, sig.strong_sum_len, err.Error())
			return
		}

		sig.block_sigs[block.wsum] = append(sig.block_sigs[block.wsum], block)
		if tag, ok = sig.tag_tables[block.wsum]; ok {
			if count > tag.l {
				tag.r = count
			}

		} else {
			sig.tag_tables[block.wsum] = &rs_tag_table_entry{count, count}
		}

		count++
		if debug {
			fmt.Printf("LoadSign: block count %d, wsum: 0x%x ssum: %s\n", count, block.wsum, string(block.ssum))
		}
	}

	if debug {
		fmt.Printf("LoadSign: block len: %d strong sum len: %d total len: %d count: %d\n",
			sig.block_len, sig.strong_sum_len, sig.flength, count)
	}

	if count == 0 {
		return
	}

	// 对sig.block_sigs排序
	for _, blocks = range sig.block_sigs {
		sort.Sort(blockSlice(blocks))
	}

	sig.count = count

	// build
	//err = buildSign(sig)

	return
}

type blockSlice []*rs_block_sig

func (s blockSlice) Len() int {
	return len(s)
}

func (s blockSlice) Less(i, j int) bool {
	c := bytes.Compare(s[i].ssum, s[j].ssum)
	if c < 0 {
		return true
	}
	if c == 0 {
		return s[i].i < s[j].i
	}
	return false
}

func (s blockSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// 查找
// 由于blockSlice的weaksum都相同(map的key相同)，search时，只需比较strongsum
// 当strongSum相同时，比较pos与各个block的pos的关系，取最小的
func (s blockSlice) search(ssum []byte, pos int64, blockLen uint32) (matchAt int64) {
	matchAt = -1
	found := -1
	bl := int64(blockLen)
	i, j := 0, len(s)
	for i < j {
		h := i + (j-i)/2 // avoid overflow when computing h
		// i ≤ h < j
		c := bytes.Compare(ssum, s[h].ssum)
		if c == 0 {
			found = h
			if int64(s[h].i)*bl == pos {
				// 精确匹配到，直接返回
				return pos
			}
			break
		} else if c > 0 {
			i = h + 1 // preserves f(j) == true
		} else {
			j = h
		}
	}

	if found == -1 {
		return
	}
	matchAt = int64(s[found].i) * int64(blockLen)
	if len(s) == 1 {
		return
	}

	var (
		idx     int
		minIdx  int = found
		minDist int64
	)
	minDist = abs(int64(s[found].i)*bl - pos)
	// 向前轮询，找到最小distance的idx
	for idx = found - 1; idx >= 0; idx-- {
		if bytes.Compare(ssum, s[idx].ssum) == 0 {
			dist := abs(int64(s[idx].i)*bl - pos)
			if dist <= minDist {
				minDist = dist
				minIdx = idx
			} else {
				break
			}
		} else {
			break
		}
	}
	// 向后轮询，找到最小distance的idx
	for idx = found + 1; idx < len(s); idx++ {
		if bytes.Compare(ssum, s[idx].ssum) == 0 {
			dist := abs(int64(s[idx].i)*bl - pos)
			//println("idx:", idx, "minDist:", minDist, "dist:", dist)
			if dist < minDist {
				minDist = dist
				minIdx = idx
			} else {
				break
			}
		} else {
			break
		}
	}
	matchAt = int64(s[minIdx].i) * int64(blockLen)

	return
}

func abs(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}
func buildSign(sig *Signature) (err error) {
	/*
		i := 0

		sig.tag_table = make(map[uint32]*rs_tag_table_entry)
		//sig.tag_table = make([]rs_tag_table_entry, TABLE_SIZE)
			sig.targets = make([]rs_target, sig.count)
			for i = 0; i < sig.count; i++ {
				sig.targets[i].i = i
				sig.targets[i].t = gettag(sig.block_sigs[i].wsum)
			}

			// 对sig.targets排序, 按照target.t从小到大排列
			sort.Sort(sig)
	*/
	// initialize
	/*
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
	*/

	return
}

/*
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
*/
