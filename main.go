package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"
	"go-safer/passes/literalheader"
)

func main() {
	singlechecker.Main(literalheader.Analyzer)
}
