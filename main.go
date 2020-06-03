package main

import (
	"github.com/jlauinger/go-safer/passes/sliceheader"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(sliceheader.Analyzer)
}
