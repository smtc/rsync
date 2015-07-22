package rsync

import (
	"github.com/dchest/blake2b"
	"github.com/smtc/rollsum"
)

// calculate weaksum (rollsum alder32)
// calculate strong sum (blake algthorithm)

// use rollsum do weak sum
func weakSum(p []byte) (s uint32) {
	var rs rollsum.Rollsum

	rs.Init()
	rs.Update(p)
	return rs.Digest()
}

// use blake do strong sum
// sumLen: 32 or 64
func strongSum(p []byte, sumLen uint32) (s []byte) {
	if sumLen == 32 {
		sum := blake2b.Sum256(p)
		s = sum[0:sumLen]
		return
	}

	sumLen = 64 // make sure sumLen is 64
	sum := blake2b.Sum512(p)
	s = sum[0:sumLen]

	return
}
