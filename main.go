package main

import (
	"flag"
	"os"
)

var (
	gopath      = os.Getenv("GOPATH")
	prevPkgFlag = flag.String("prev", "", "")
	newPkgFlag  = flag.String("new", "", "")
	outputFlag  = flag.String("output", "", "")
)

func main() {
	flag.Parse()
	// log.Println("parsing packages")
	// changes, err := parse(*prevPkgFlag, *newPkgFlag)
	// if err != nil {
	// 	panic(err)
	// }
}
