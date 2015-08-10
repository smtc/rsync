package main

import (
	"bytes"
	"fmt"
	"io"
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
	app.Usage = "rdiff   signature BASIS [SIGNATURE]\n" +
		"               delta [OPTIONS] SIGNATURE NEWFILE [DELTA]\n" +
		"               patch BASIS DELTA [NEWFILE]\n"

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
			Usage:   "Signature generate use blake2 algorithm\n",
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
				"  -b, --block-size=BYTES    Signature block size\n" +
				"  -S, --sum-size=BYTES      Set signature strength\n",
			Action: doDelta,
		},
		{
			Name:    "patch",
			Aliases: []string{"p"},
			Usage:   "complete a task on the list",
			Action:  doPatch,
		},
	}
}

func doSign(c *cli.Context) {
	var (
		fn     string
		fnLen  int64
		outFn  string
		st     os.FileInfo
		inRd   *os.File
		outWr  io.Writer
		err    error
		stdout bool
	)
	fmt.Println(c.Int("block-size"), c.Int("sum-size"), c.Bool("verbose"))
	return
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
	if st, err = inRd.Stat(); err != nil {
		fmt.Println("Stat", fn, "failed:", err)
		return
	} else {
		fnLen = st.Size()
	}
	// 输出文件, 如果没有提供，使用stdout
	if args == 2 {
		outFn = c.Args().Get(1)
		if outWr, err = os.Open(outFn); err != nil {
			fmt.Println("Open", outFn, "failed:", err)
			return
		}
	} else {
		stdout = true
		outWr = bytes.NewBuffer([]byte{})
	}

	err = rsync.GenSign(inRd,
		fnLen,
		uint32(c.Int("block-size")),
		uint32(c.Int("block-size")),
		outWr)
	if err != nil {
		fmt.Println("Generate signature failed:", err)
		return
	}

	if stdout {
	}
}

func doDelta(c *cli.Context) {

}

func doPatch(c *cli.Context) {

}
