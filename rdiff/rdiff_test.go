package main

import (
	//"os"
	"testing"
)

func TestCli(t *testing.T) {
	app := setupApp()
	app.Run([]string{"rsync", "--verbose", "signature"})
	app.Run([]string{"rsync", "-verbose", "signature"})
	app.Run([]string{"rsync", "signature", "a.txt"})
}

func TestRdiff(t *testing.T) {

}
