package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/keep94/mailmerge/merge"
	"github.com/keep94/toolbox/build"
)

var (
	fCsv     string
	fNoGo    string
	fVersion bool
)

func main() {
	flag.Parse()
	if fVersion {
		version, _ := build.MainVersion()
		fmt.Println(build.BuildId(version))
		return
	}
	if fCsv == "" || fNoGo == "" {
		fmt.Println("-csv, and -nogo flags required.")
		flag.Usage()
		os.Exit(2)
	}
	csvFile, err := merge.ReadCsv(fCsv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := csvFile.SelectGoing().WithNotGoing().Write(fNoGo); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&fCsv, "csv", "", "Path to source CSV file")
	flag.StringVar(&fNoGo, "nogo", "", "Path to nogo CSV file being created")
	flag.BoolVar(&fVersion, "version", false, "Show version")
}
