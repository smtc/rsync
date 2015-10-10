# rsync

This is rsync libary in pure golang.

# Signature

    func GenSign(rd io.Reader, rdLen int64, blockLen uint32, result io.Writer) (err error)

generate signature for rd.

# Delta

    func GenDelta(dstSig io.Reader, src io.ReadSeeker, srcLen int64, result io.Writer) (err error)

generate delta, and delta will write to result.

# Patch

    func Patch(deltaRd io.Reader, target io.ReadSeeker, merged io.Writer, args ...bool) (err error)

patch.

# rdiff

