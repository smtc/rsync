package rsync

import (
	"fmt"
	"io/ioutil"
	"os"
	//"path"
	"path/filepath"
	"testing"
)

func testFuzz(t *testing.T) {
	var blocks = []int{2, 4, 8, 16}

	filepath.Walk("./testdata/corpus/",
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				t.Logf("path %s is dir!\n", path)
				return nil
			}
			if err != nil {
				t.Logf("Walk file %s failed: %s\n", path, err.Error())
				return err
			}
			t.Logf("review %s:\n", path)
			content, ierr := ioutil.ReadFile(path)
			if ierr != nil {
				t.Logf("read file %s failed: %s\n", path, ierr.Error())
				return nil
			}

			// split by random
			for i := 1; i < len(content); i++ {
				src := content[0:i]
				dst := content[i:len(content)]
				for _, b := range blocks {
					ret := doFuzz(path, src, dst, b, true)
					if ret != 1 {
						t.Fail()
					}
				}
			}
			return nil
		})
}

// path: file path
// pos: 从文件的什么位置把文件分成两部分，分别作为src和dst
func doFuzzBytes(t *testing.T, path string, content []byte, pos int, blocks []int, debug bool) {
	src := content[0:pos]
	dst := content[pos:len(content)]
	for _, b := range blocks {
		ret := doFuzz(path, src, dst, b, debug)
		if ret != 1 {
			t.Fail()
		}
	}

}

func TestFuzzManual(t *testing.T) {
	var blocks = []int{2, 8, 16, 64, 256, 1024, 2048, 4196}

	filepath.Walk("./testdata/corpus/",
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				t.Logf("path %s is dir!\n", path)
				return nil
			}
			if err != nil {
				t.Logf("Walk file %s failed: %s\n", path, err.Error())
				return err
			}
			fmt.Printf("review %s:\n", path)
			content, ierr := ioutil.ReadFile(path)
			if ierr != nil {
				t.Logf("read file %s failed: %s\n", path, ierr.Error())
				return nil
			}
			content = append(content, content...)

			// split by random
			for i := 0; i <= len(content); i++ {
				src := content[0:i]
				dst := content[i:len(content)]
				for _, b := range blocks {
					fmt.Printf("  block=%d len(src)=%d len(dst)=%d i=%d\n",
						b, len(src), len(dst), i)
					ret := doFuzz(path, src, dst, b, false)
					if ret != 1 {
						t.Fail()
					}
				}
			}
			return nil
		})
}
