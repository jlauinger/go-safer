package main

import (
	"github.com/jlauinger/go-safer/passes/sliceheader"
	"github.com/jlauinger/go-safer/passes/structcast"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(sliceheader.Analyzer, structcast.Analyzer)
}
