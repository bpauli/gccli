package main

import (
	"fmt"
	"os"

	"github.com/bpauli/gccli/conformance"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: scrub <record dir> <out dir>")
		os.Exit(2)
	}
	if err := conformance.ScrubDir(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
