package rsync

import (
	"bytes"
	"compress/gzip"
	"testing"
)

// gzip压缩后的字符串最小23个字节，当字符串本身42个字节时，才与压缩后的字符串长度相等
// 因此，考虑到gzip本身的消耗，我认为，长度超过64个字节的字符串压缩才有意义，否则，意义不大
func testGzip(t *testing.T) {
	var b *bytes.Buffer

	for _, s := range ss {
		b = bytes.NewBuffer([]byte(""))
		w := gzip.NewWriter(b)
		w.Write([]byte(s))
		w.Close()
		t.Logf("before compress: len(s)=%d after: %d\n", len(s), len(b.Bytes()))
	}
	for _, s := range chs {
		b = bytes.NewBuffer([]byte(""))
		w := gzip.NewWriter(b)
		w.Write([]byte(s))
		w.Close()
		t.Logf("before compress: len(s)=%d after: %d\n", len(s), len(b.Bytes()))
	}
}
