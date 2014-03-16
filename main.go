package main

import (
	"flag"
	"os"
	"github.com/bom-d-van/goutils/gocheckutils/logutils"

	"github.com/bom-d-van/vermouth/verparser"
)

var (
	prevPkgFlag = flag.String("prev", "", "package path to old version of the package")
	newPkgFlag  = flag.String("new", "", "package path to new version of the package")
	outputFlag  = flag.String("output", "", "print out changes to")
	debugFlag   = flag.Bool("debug", false, "print out debug logs")
)

func main() {
	flag.Parse()
	if !*debugFlag {
		logutils.NullLogOutput()
	}
	if *prevPkgFlag == "" || *newPkgFlag == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	changes, err := verparser.Parse(*prevPkgFlag, *newPkgFlag)
	if err != nil {
		panic(err)
	}
	if *outputFlag != "" {

	} else {
		os.Stdout.Write([]byte(changes.GenDoc()))
	}
}
