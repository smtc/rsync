package main

import (
	"path"
	//"bytes"
	"fmt"
	//"io"
	"os"

	"github.com/codegangsta/cli"
	"github.com/smtc/rsync"
)

const (
	version = "0.1.0"
)

func main() {

	app := setupApp()

	app.Run(os.Args)
}

func setupApp() *cli.App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "rdiff"
	app.Version = version
	app.Usage = "    signature [OPTIONS] BASIS [SIGNATURE]\n" +
		"               delta [OPTIONS] SIGNATURE NEWFILE [DELTA]\n" +
		"               patch [OPTIONS] BASIS DELTA [NEWFILE]\n"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "verbose",
			EnvVar: "verbose",
			Usage:  "debug rsync details",
		},
	}
	setupCommands(app)

	return app
}

func setupCommands(app *cli.App) {
	app.Commands = []cli.Command{
		{
			Name:    "signature",
			Aliases: []string{"s"},
			Usage: "Signature generate use blake2 algorithm\n" +
				"     -b, --block-size=BYTES    Signature block size\n" +
				"     -s, --sum-size=BYTES      Set signature strength\n",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "block-size,b",
					Value: 2048,
					Usage: "Set signature block size",
				},
				cli.IntFlag{
					Name:  "sum-size,s",
					Value: 32,
					Usage: "Set signature strong checksum strength, 32 or 64",
				},
			},
			Action: doSign,
		},
		{
			Name:    "delta",
			Aliases: []string{"d"},
			Usage: "Delta-encoding options:\n" +
				"     -b, --block-size=BYTES    Signature block size\n" +
				"     -s, --sum-size=BYTES      Set signature strength\n",

			Action: doDelta,
		},
		{
			Name:    "patch",
			Aliases: []string{"p"},
			Usage: "complete a task on the list\n" +
				"     -b, --block-size=BYTES    Signature block size\n" +
				"     -s, --sum-size=BYTES      Set signature strength\n",

			Action: doPatch,
		},
	}
}

// rdiff signature [-b {block_size}] [-s {sum_size}] {basic_file} [delta_file]
func doSign(c *cli.Context) {
	var (
		err   error
		fn    string
		fnLen int64
		outFn string
		st    os.FileInfo
		inRd  *os.File
		outWr *os.File
	)

	args := len(c.Args())
	if args == 0 || args > 2 {
		fmt.Println("No param found or too many params.\nUsage:", c.App.Usage)
		return
	}
	// basic文件
	fn = c.Args().First()
	if inRd, err = os.Open(fn); err != nil {
		fmt.Println("Open", inRd, "failed:", err)
		return
	}
	// close
	defer inRd.Close()
	if st, err = inRd.Stat(); err != nil {
		fmt.Println("Stat", fn, "failed:", err)
		return
	} else {
		fnLen = st.Size()
	}
	// 输出文件, 如果没有提供，使用stdout
	if args == 2 {
		outFn = c.Args().Get(1)
	} else {
		outFn = c.Args().Get(0) + ".sign"
	}
	// 打开输出文件
	if outWr, err = os.OpenFile(outFn, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm); err != nil {
		fmt.Println("Open", outFn, "failed:", err)
		return
	}
	defer outWr.Close()

	err = rsync.GenSign(inRd,
		fnLen,
		uint32(c.Int("block-size")),
		uint32(c.Int("block-size")),
		outWr)
	if err != nil {
		fmt.Println("Generate signature failed:", err)
		return
	}
}

// rdiff delta {signature_file} {source_file} [delta_file]
func doDelta(c *cli.Context) {
	var (
		err    error
		fn     string
		srcLen int64
		srcFn  string
		outFn  string
		fi     os.FileInfo
		signRd *os.File
		srcRd  *os.File
		outWr  *os.File
	)

	args := len(c.Args())
	if args < 2 || args > 3 {
		fmt.Println("No param found or too many params.\nUsage:", c.App.Usage)
		return
	}
	// signature文件
	fn = c.Args().First()
	srcFn = c.Args().Get(1)
	if args == 3 {
		outFn = c.Args().Get(2)
	} else {
		outFn = srcFn + "-delta"
	}

	// open & close signature file
	if signRd, err = os.Open(fn); err != nil {
		fmt.Printf("Open signature file %s failed: %v\n", fn, err)
		return
	}
	defer signRd.Close()

	// open & close source file
	if srcRd, err = os.Open(srcFn); err != nil {
		fmt.Printf("open source file %s failed: %v\n", srcFn, err)
		return
	}
	defer srcRd.Close()

	if fi, err = srcRd.Stat(); err != nil {
		fmt.Printf("stat source file %s failed: %v\n", srcFn, err)
		return
	}
	srcLen = fi.Size()

	// open & close delta file
	if outWr, err = os.OpenFile(outFn, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm); err != nil {
		fmt.Printf("open delta file %s failed: %v\n", srcFn, err)
		return
	}
	defer outWr.Close()

	err = rsync.GenDelta(signRd, srcRd, srcLen, outWr)
	if err != nil {
		fmt.Printf("generate delta file %s failed: %v\n", outFn, err)
	}
}

// rdiff patch {delta_file} {dest_file} [result_file]
func doPatch(c *cli.Context) {
	var (
		err error
		fn  string

		destFn  string
		outFn   string
		deltaRd *os.File
		destRd  *os.File
		outWr   *os.File
	)

	args := len(c.Args())
	if args < 2 || args > 3 {
		fmt.Println("No param found or too many params.\nUsage:", c.App.Usage)
		return
	}

	// delta文件
	fn = c.Args().First()
	destFn = c.Args().Get(1)
	if args == 3 {
		outFn = c.Args().Get(2)
	} else {
		ext := path.Ext(destFn)
		outFn = destFn[0:len(destFn)-len(ext)] + "-patch" + ext
	}

	// open & close signature file
	if deltaRd, err = os.Open(fn); err != nil {
		fmt.Printf("Open signature file %s failed: %v\n", fn, err)
		return
	}
	defer deltaRd.Close()

	// open & close source file
	if destRd, err = os.Open(destFn); err != nil {
		fmt.Printf("open source file %s failed: %v\n", destFn, err)
		return
	}
	defer destRd.Close()

	err = rsync.Patch(deltaRd, destRd, outWr)
	if err != nil {
		fmt.Printf("patch file %s failed: %v\n", outFn, err)
	}
}
