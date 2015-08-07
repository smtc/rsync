package rsync

import (
	"bytes"
	"fmt"
	"testing"
)

func TestIntLength(t *testing.T) {
	Assert(int64Length(0x100000000) == 8, "0x100000000 length should be 8")
	Assert(int64Length(0x10000000) == 4, "0x10000000 length should be 4")
	Assert(int64Length(0x1000) == 2, "0x1000 length should be 2")
	Assert(int64Length(0x10) == 1, "0x10 length should be 1")

	Assert(intLength(0x10000000) == 4, "0x10000000 should be 4")
	Assert(intLength(0x1000) == 2, "0x1000 should be 2")
	Assert(intLength(0x10) == 1, "0x10 should be 1")
}

/*
测试以下情况
1 完全相同的情况
  1.1 各种长度与blockLen的不同关系

2 完全没有相同部分的情况

3 dst是src的子集
  3.1 dst在src中的位置
  3.2 dst长度与blockLen的不同关系
  3.3 src与blockLen的不同关系

4 src是dst子集
  4.1 src在dst中的位置
  4.2 dst长度与blockLen的不同关系
  4.3 src与blockLen的不同关系

5
*/
func TestDelta(t *testing.T) {
	var b []uint32 = []uint32{1, 2, 3, 4, 5}

	testAllSame(t, b)
	//testAllDiff(t, b)
	//testSrcSubDst(t, b)
	//testDstSubSrc(t, b)
	//testCrossDiff(t, b)

}

func testStringDelta(t *testing.T, src, dst string, bl []uint32) (dfs []*delta, err error) {
	var (
		df     *delta
		srcSig *bytes.Buffer
		result *bytes.Buffer
		srcRd  *bytes.Reader
		dstRd  *bytes.Reader
	)

	fmt.Printf("\ntestStingDelta: src=%s dst=%s\n", src, dst)
	for _, blen := range bl {
		t.Logf("blockLen: %d src: %s dst: %s\n", blen, string(src), string(dst))
		srcSig = bytes.NewBuffer([]byte(""))
		result = bytes.NewBuffer([]byte(""))
		srcRd = bytes.NewReader([]byte(src))
		dstRd = bytes.NewReader([]byte(dst))
		df = new(delta)

		err = GenSign(srcRd, int64(len(src)), 32, blen, srcSig)
		if err != nil {
			t.Fatal("GenSign failed:", err)
			return
		}

		srcSig = bytes.NewBuffer(srcSig.Bytes())
		if df.sig, err = LoadSign(srcSig); err != nil {
			t.Fatal("LoadSign failed:", err)
			return
		}

		df.blockLen = blen
		df.outer = result
		if err = df.genDelta(dstRd, int64(len(dst))); err != nil {
			t.Fatal("genDelta failed:", err)
			return
		}
		df.dumpMatchStats(result)

		t.Log("delta Match/Miss:\n", string(result.Bytes()))

		dfs = append(dfs, df)
	}

	for i := 0; i < len(dfs)-1; i++ {
		df1 := dfs[i]
		df2 := dfs[i+1]

		if df1.equalMatchStats(df2) == false {
			t.Fatalf("shoud be equal!")
		}
	}
	return
}

// 1 完全相同的情况
func testAllSame(t *testing.T, bl []uint32) {
	var (
		err error
	)
	for _, s := range ss {
		t.Logf("blockLen: %d src: %s dst: %s\n", bl, s, s)
		_, err = testStringDelta(t, s, s, bl)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// 2 完全不同
func testAllDiff(t *testing.T, bl []uint32) {
	var (
		err error
		ds  []string = []string{
			"1",
			"12",
			"123",
			"1234",
			"12345",
			"123456",
			"1234567",
			"12345678",
			"123456789",
			"1234567890",
		}
		ss []string = []string{
			"",
			"a",
			"ab",
			"abc",
			"abcd",

			"abcde",
			"abcdef",
			"abcdefg",
			"abcdefgh",
			"abcdefghi",
			"abcdefghij",
			"abcdefghijk",
			"abcdefghijkl",
			"abcdefghijklm",
			"abcdefghijklmn",
		}
	)
	for _, s1 := range ds {
		for _, s2 := range ss {
			t.Logf("blockLen: %d src: %s dst: %s\n", bl, s1, s2)
			_, err = testStringDelta(t, s1, s2, bl)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

}

func testSrcSubDst(t *testing.T, bl []uint32) {
	var (
		err    error
		sl     = len(ss)
		s1, s2 string
	)
	for i := 0; i < sl; i++ {
		s1 = ss[i]
		for j := 0; j < sl; j++ {
			s2 = ss[j]
			_, err = testStringDelta(t, s1, s2, bl)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func testDstSubSrc(t *testing.T, bl []uint32) {

}

func testCrossDiff(t *testing.T, bl []uint32) {

}
