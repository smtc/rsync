package rsync

import (
	"bytes"
	"log"
	"testing"
)

func testPatch1(t *testing.T) {
	var (
		s1          = "abcdefghijklmn"
		s2          = "dejkopq"
		bl []uint32 = []uint32{1, 2, 3, 4, 5}
	)

	_ = s1
	_ = s2
	_ = bl
	testPatchString(t, 2, "12", "123")
	testPatchString(t, 2, "123", "12")

	for i := 0; i < len(ss)+len(chs); i++ {
		if i < len(ss) {
			s1 = ss[i]
		} else {
			s1 = chs[i-len(ss)]
		}
		for j := 0; j < len(ss)+len(chs); j++ {
			if j < len(ss) {
				s2 = ss[j]
			} else {
				s2 = chs[j-len(ss)]
			}
			for _, l := range bl {
				testPatchString(t, l, s1, s2)
			}
		}
	}

	/*
		for _, l := range bl {
			s := testPatchString(t, l, s1, s2)
			t.Log("patch result:", s)
		}
	*/
}

func testPatch2(t *testing.T) {
	var (
		s1, s2 string
		bl     []uint32 = []uint32{1, 2, 3, 4, 5}
		src    string   = "abcdefghijklmnopqrstuvwxyz1234567890"
		dsts   []string = []string{
			"ac",
			"a0",
			"akl",
			"aklm",
			"a7890",
			"f7890",
			"abl",
			"abpq",
			"ab90",
			"ab890",
			"bc7890",
			"depqrstu",
			"ab67890",
			"fgw",
			"nrst",
			"oqr",
			"xz1",
			"xyz7890",
			"abcd7890",
			"acfi",
			"abklpquv",
		}
	)

	s1 = src
	for j := 0; j < len(dsts); j++ {
		s2 = dsts[j]

		for _, l := range bl {
			testPatchString(t, l, s1, s2)
			testPatchString(t, l, s2, s1)
		}
	}
}

func testPatchString(t *testing.T, bl uint32, src, dst string) (result string) {
	var err error

	log.Printf("testPatchString: bl=%d src=%s dst=%s\n", bl, src, dst)
	srcRd := bytes.NewReader([]byte(src))
	sigWr := bytes.NewBuffer([]byte(""))
	if err = GenSign(srcRd, int64(len(src)), 32, bl, sigWr); err != nil {
		panic("gen sign failed")
	}
	deltWr := bytes.NewBuffer([]byte(""))
	dstRd := bytes.NewReader([]byte(dst))
	if err = GenDelta(sigWr, dstRd, int64(len(dst)), deltWr, true); err != nil {
		panic("gen delta failed" + err.Error() + ": src: " + src + " dst: " + dst)
	}
	srcRd = bytes.NewReader([]byte(src))
	log.Printf("testPatchString: bl=%d src=%s dst=%s %x \n", bl, src, dst, deltWr.Bytes())
	deltRd := bytes.NewBuffer(deltWr.Bytes())
	merged := bytes.NewBuffer([]byte(""))
	if err = Patch(deltRd, srcRd, merged); err != nil {
		panic("patch failed!")
	}
	result = string(merged.Bytes())
	if result != dst {
		t.Failed()
	}
	return
}
