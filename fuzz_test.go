package rsync

import (
	"io/ioutil"
	"os"
	//"path"
	"path/filepath"
	"testing"
)

func TestFuzz(t *testing.T) {
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
					ret := doFuzz(src, dst, b, true)
					if ret != 1 {
						t.Fail()
					}
				}
			}
			return nil
		})
}
