package main

import (
	"go-safer/passes/sliceheader"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(sliceheader.Analyzer)
}
