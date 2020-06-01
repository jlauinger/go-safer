package sliceheader_test

import (
	"go-safer/passes/sliceheader"
	"golang.org/x/tools/go/analysis/analysistest"
	"testing"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	testPackages := []string{
		"bad/composite_literal",
		"bad/composite_in_composite",
		"bad/header_in_struct",
		"bad/type_alias",
		"bad/variable_declaration",
		"bad/unsafe_cast",
		"bad/nil_cast",

		"good/safe_cast",
		"good/safe_cast_dereferenced_header",
		"good/unrelated_selector",
	}
	analysistest.Run(t, testdata, sliceheader.Analyzer, testPackages...)
}

