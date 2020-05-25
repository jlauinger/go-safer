package literalheader_test

import (
	"testing"
  "go-safer/passes/literalheader"
	"golang.org/x/tools/go/analysis/analysistest"
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
		"good/unrelated_selector",
	}
	analysistest.Run(t, testdata, literalheader.Analyzer, testPackages...)
}

