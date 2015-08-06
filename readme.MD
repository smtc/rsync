# rsync

This is rsync in golang.

# to be continued ....

1 delta need test

2 patch not implement yet

3 need a command line tool like rdiff

# Signature

    func GenSign(rd io.Reader, rdLen int64, sumLen, blockLen uint32, result io.Writer) (err error)

# Delta

    func GenDelta(dstSig io.Reader, src io.ReadSeeker, srcLen int64, result io.Writer) (err error)

# Patch

# rdiff

