package rsync

import (
	"fmt"
	"sort"
	"testing"
)

var (
	rs1 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{1, 5439, []byte("1234")},
	}

	rs20 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{2, 5439, []byte("1234")},
		&rs_block_sig{1, 5439, []byte("1234")},
	}
	rs21 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{1, 5439, []byte("1234")},
		&rs_block_sig{2, 5439, []byte("1234")},
	}

	rs22 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{1, 5439, []byte("1236")},
		&rs_block_sig{2, 5439, []byte("1235")},
	}
	rs23 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{1, 5439, []byte("1236")},
		&rs_block_sig{2, 5439, []byte("1235")},
	}
	rs30 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{3, 5439, []byte("1234")},
		&rs_block_sig{1, 5439, []byte("1234")},
		&rs_block_sig{2, 5439, []byte("1234")},
	}
	rs31 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{2, 5439, []byte("12234")},
		&rs_block_sig{3, 5439, []byte("01234")},
		&rs_block_sig{1, 5439, []byte("12364")},
	}
	rs32 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{2, 5439, []byte("12345")},
		&rs_block_sig{3, 5439, []byte("1234")},
		&rs_block_sig{1, 5439, []byte("1234")},
	}
	rs33 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{3, 5439, []byte("12345")},
		&rs_block_sig{2, 5439, []byte("12345")},
		&rs_block_sig{1, 5439, []byte("1234")},
	}

	rs40 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{2, 5439, []byte("1234")},
		&rs_block_sig{1, 5439, []byte("1234")},
		&rs_block_sig{4, 5439, []byte("1234")},
		&rs_block_sig{3, 5439, []byte("1234")},
	}

	rs41 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{2, 5439, []byte("1234")},
		&rs_block_sig{1, 5439, []byte("1234")},
		&rs_block_sig{4, 5439, []byte("123sf4")},
		&rs_block_sig{3, 5439, []byte("123ee4")},
		&rs_block_sig{5, 5439, []byte("1232ddsa4")},
	}
	rs42 []*rs_block_sig = []*rs_block_sig{
		&rs_block_sig{8, 5439, []byte("1234")},
		&rs_block_sig{2, 5439, []byte("1234")},
		&rs_block_sig{1, 5439, []byte("1234")},
		&rs_block_sig{4, 5439, []byte("123sf4")},
		&rs_block_sig{9, 5439, []byte("123sf4")},
		&rs_block_sig{10, 5439, []byte("123sf4")},
		&rs_block_sig{11, 5439, []byte("123sf4")},
		&rs_block_sig{3, 5439, []byte("123ee4")},
		&rs_block_sig{5, 5439, []byte("1232ddsa4")},
		&rs_block_sig{6, 5439, []byte("1234")},
		&rs_block_sig{7, 5439, []byte("1234")},
	}
)

func TestSort(t *testing.T) {
	testSignSort(t)
	testSignSearch(t)
}

func sortBlocks(rs []*rs_block_sig) {
	sort.Sort(blockSlice(rs))
	/*
		for _, item := range rs {
			fmt.Print("{", item.i, item.wsum, " ", string(item.ssum), "} ")
		}
		fmt.Println()
	*/
}

func testSignSort(t *testing.T) {
	sortBlocks(rs1)

	sortBlocks(rs20)
	sortBlocks(rs21)
	sortBlocks(rs22)
	sortBlocks(rs23)

	sortBlocks(rs30)
	sortBlocks(rs31)
	sortBlocks(rs32)
	sortBlocks(rs33)

	sortBlocks(rs40)
	sortBlocks(rs41)
	sortBlocks(rs42)
}

func searchBlocks(rs []*rs_block_sig, sum []byte, pos int64, answer int64, name string) {
	matchAt := blockSlice(rs).search(sum, pos, 2048)
	if matchAt != answer {
		fmt.Printf("search in %s failed: return: %d should: %d, pos: %d\n", name, matchAt, answer, pos)
	}
}
func testSignSearch(t *testing.T) {
	searchBlocks(rs1, []byte("1234"), 2048, 2048, "rs1")
	searchBlocks(rs1, []byte("1234"), 0, 2048, "rs1")
	searchBlocks(rs1, []byte("1234"), 20480, 2048, "rs1")
	searchBlocks(rs1, []byte("1235"), 20480, -1, "rs1")

	searchBlocks(rs20, []byte("1234"), 20480, 4096, "rs20")
	searchBlocks(rs20, []byte("1234"), 2048, 2048, "rs20")
	searchBlocks(rs20, []byte("1234"), 3071, 2048, "rs20")
	searchBlocks(rs20, []byte("1234"), 3072, 2048, "rs20")
	searchBlocks(rs20, []byte("1234"), 3073, 4096, "rs20")
	searchBlocks(rs20, []byte("1235"), 20480, -1, "rs20")

	searchBlocks(rs22, []byte("1236"), 1111, 2048, "rs22")
	searchBlocks(rs22, []byte("1235"), 1111, 4096, "rs22")

	searchBlocks(rs30, []byte("1234"), 1111, 2048, "rs30")
	searchBlocks(rs30, []byte("1234"), 2000, 2048, "rs30")
	searchBlocks(rs30, []byte("1234"), 3071, 2048, "rs30")
	searchBlocks(rs30, []byte("1234"), 3072, 2048, "rs30")
	searchBlocks(rs30, []byte("1234"), 3073, 4096, "rs30")
	searchBlocks(rs30, []byte("1234"), 4097, 4096, "rs30")
	searchBlocks(rs30, []byte("1234"), 5120, 4096, "rs30")
	searchBlocks(rs30, []byte("1234"), 5121, 6144, "rs30")

	searchBlocks(rs42, []byte("1234"), 100, 2048, "rs42")
	searchBlocks(rs42, []byte("1234"), 10000, 2048*6, "rs42")
	searchBlocks(rs42, []byte("1234"), 2048*6+1024, 2048*6, "rs42")
	searchBlocks(rs42, []byte("1234"), 2048*6+1025, 2048*7, "rs42")
	searchBlocks(rs42, []byte("10234"), 2048*6+1025, -1, "rs42")
	searchBlocks(rs42, []byte("123sf4"), 2048*4+1025, 2048*4, "rs42")
	searchBlocks(rs42, []byte("123sf4"), 2048*8-1024, 2048*9, "rs42")
	searchBlocks(rs42, []byte("123sf4"), 2048*9+1024, 2048*9, "rs42")
	searchBlocks(rs42, []byte("123sf4"), 2048*9+1025, 2048*10, "rs42")
}
