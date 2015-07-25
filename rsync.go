package rsync

import (
//	"io"
)

const (
	DeltaMagic uint32 = 0x72730236
	BlakeMagic uint32 = 0x72730137
	Md4Magic   uint32 = 0x72730136

	TABLE_SIZE = (1 << 16)
	NULL_TAG   = -1
)

/*
 * This structure describes all the sums generated for an instance of
 * a file.  It incorporates some redundancy to make it easier to
 * search.
 */
type Signature struct {
	flength        int64  /* total file length */
	count          int    /* how many chunks */
	remainder      int    /* flength % block_length */
	block_len      uint32 /* block_length */
	strong_sum_len uint32
	block_sigs     map[uint32][]*rs_block_sig /* points to info for each chunk */
	tag_tables     map[uint32]*rs_tag_table_entry
	//tag_table      []rs_tag_table_entry
	//targets        []rs_target
	magic uint32
}

// from librsync sunset.h

// signature target
type rs_target struct {
	t uint16
	i int
}

type rs_tag_table_entry struct {
	l int // left bound of the hash tag in sorted array of targets
	r int // right bound of the hash tag in sorted array of targets
	// all tags between l and r inclusively are the same
}

type rs_block_sig struct {
	i    int    /* index of this chunk */
	wsum uint32 /* simple checksum */
	ssum []byte /* strong checksum  */
}
