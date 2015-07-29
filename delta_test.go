package rsync

import (
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
