package structcast

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is a golang.org/x/tools/go/analysis style linter pass.
// Use this with the Vet-style infrastructure.
var Analyzer = &analysis.Analyzer{
	Name:             "structcast",
	Doc:              "reports unsafe struct casts where the target struct contains architecture sized variables",
	Run:              run,
	Requires:         []*analysis.Analyzer{inspect.Analyzer, ctrlflow.Analyzer},
	RunDespiteErrors: true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	inspectResult.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		node := n.(*ast.CallExpr)
		if structCastWithMismatchingTargetLength(node, pass) {
			pass.Reportf(n.Pos(), "unsafe cast between structs with mismatching count of platform dependent field sizes")
		}
	})

	return nil, nil
}

func structCastWithMismatchingTargetLength(expr *ast.CallExpr, pass *analysis.Pass) bool {
	src, dst, ok := detectUnsafeCast(expr)
	if !ok {
		return false
	}

	srcType := getObjectType(src, pass)
	dstType := getObjectType(dst, pass)

	return checkIncompatibleStructsCast(srcType, dstType)
}

func detectUnsafeCast(expr ast.Expr) (ast.Expr, ast.Expr, bool) {
	var castExpr ast.Expr
	starExpr, ok := expr.(*ast.StarExpr)
	if ok {
		castExpr = starExpr.X
	} else {
		castExpr = expr
	}

	targetCastCallExpr, ok := castExpr.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}

	if len(targetCastCallExpr.Args) == 0 {
		return nil, nil, false
	}

	sourceCastCallExpr, ok := targetCastCallExpr.Args[0].(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}

	sourceCastUnsafePointerSelector, ok := sourceCastCallExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}

	sourceCastUnsafePointerSelectorX, ok := sourceCastUnsafePointerSelector.X.(*ast.Ident)

	if sourceCastUnsafePointerSelector.Sel.Name != "Pointer" || sourceCastUnsafePointerSelectorX.Name != "unsafe" {
		return nil, nil, false
	}

	var sourceExpr ast.Expr
	sourceUnary, ok := sourceCastCallExpr.Args[0].(*ast.UnaryExpr)
	if ok && sourceUnary.Op == token.AND {
		sourceExpr = sourceUnary.X
	} else {
		sourceExpr = sourceCastCallExpr.Args[0]
	}

	targetParen, ok := targetCastCallExpr.Fun.(*ast.ParenExpr)
	if !ok {
		return nil, nil, false
	}

	targetStar, ok := targetParen.X.(*ast.StarExpr)
	if !ok {
		return nil, nil, false
	}

	return sourceExpr, targetStar.X, true
}

func getObjectType(expr ast.Expr, pass *analysis.Pass) types.Type {
	return pass.TypesInfo.Types[expr].Type.Underlying()
}

func checkIncompatibleStructsCast(src types.Type, dst types.Type) bool {
	srcStruct, ok := src.(*types.Struct)
	if !ok {
		return false
	}

	dstStruct, ok := dst.(*types.Struct)
	if !ok {
		return false
	}

	srcPlatformDependentCount := 0
	dstPlatformDependentCount := 0

	for i := 0; i < srcStruct.NumFields(); i++ {
		if isPlatformDependent(srcStruct.Field(i)) {
			srcPlatformDependentCount += 1
		}
	}

	for i := 0; i < dstStruct.NumFields(); i++ {
		if isPlatformDependent(dstStruct.Field(i)) {
			dstPlatformDependentCount += 1
		}
	}

	return srcPlatformDependentCount != dstPlatformDependentCount
}

func isPlatformDependent(field *types.Var) bool {
	basicType, ok := field.Type().(*types.Basic)
	if !ok {
		return false
	}

	kind := basicType.Kind()

	if kind == types.Int || kind == types.Uint || kind == types.Uintptr {
		return true
	}

	return false
}