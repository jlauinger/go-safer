package structcast_test

import (
	"github.com/jlauinger/go-safer/passes/structcast"
	"golang.org/x/tools/go/analysis/analysistest"
	"testing"
)

func Test(t *testing.T) {
	// use go vet infrastructure testing and supply annotated code examples
	testdata := analysistest.TestData()
	testPackages := []string{
		"bad/architecture_sized_variable",

		"good/strictly_sized_struct",
		"good/no_cast",
	}
	analysistest.Run(t, testdata, structcast.Analyzer, testPackages...)
}

