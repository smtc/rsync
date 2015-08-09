package rsync

import (
	"bytes"
	"testing"
)

func TestPatch(t *testing.T) {
	var (
		s1          = "abcdefghijklmn"
		s2          = "dejkopq"
		bl []uint32 = []uint32{1} //, 2, 3, 4, 5}
	)

	for _, l := range bl {
		s := testPatchString(t, l, s1, s2)
		t.Log("patch result:", s)
	}
}

func testPatchString(t *testing.T, bl uint32, src, dst string) (result string) {
	var err error

	srcRd := bytes.NewReader([]byte(src))
	sigWr := bytes.NewBuffer([]byte(""))
	if err = GenSign(srcRd, int64(len(src)), 32, bl, sigWr); err != nil {
		panic("gen sign failed")
	}
	deltWr := bytes.NewBuffer([]byte(""))
	dstRd := bytes.NewReader([]byte(dst))
	if err = GenDelta(sigWr, dstRd, int64(len(dst)), deltWr); err != nil {
		panic("gen delta failed")
	}
	srcRd = bytes.NewReader([]byte(src))
	t.Logf("%x ", deltWr.Bytes())
	deltRd := bytes.NewBuffer(deltWr.Bytes())
	merged := bytes.NewBuffer([]byte(""))
	if err = Patch(deltRd, srcRd, merged); err != nil {
		panic("patch failed!")
	}
	result = string(merged.Bytes())
	return
}
