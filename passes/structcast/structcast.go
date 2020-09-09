package structcast

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is a golang.org/x/tools/go/analysis style linter pass.
// Use this with the Vet-style infrastructure.
var Analyzer = &analysis.Analyzer{
	Name:             "structcast",
	Doc:              "reports unsafe struct casts where the target struct contains architecture sized variables",
	Run:              run,
	Requires:         []*analysis.Analyzer{inspect.Analyzer},
	RunDespiteErrors: true,
}

/**
 * run is the entry point to the analysis pass
 */
func run(pass *analysis.Pass) (interface{}, error) {
	// get results from required inspect analyzer
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// filter AST of package under analysis for CallExpr nodes
	inspectResult.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		// check if the node is a misuse
		node := n.(*ast.CallExpr)
		if structCastWithMismatchingTargetLength(node, pass) {
			// if so, report a warning
			pass.Reportf(n.Pos(), "unsafe cast between structs with mismatching count of platform dependent field sizes")
		}
	})

	return nil, nil
}

/**
 * checks if a CallExpr node is a misused cast between incompatible architecture-dependent struct types
 */
func structCastWithMismatchingTargetLength(expr *ast.CallExpr, pass *analysis.Pass) bool {
	// first, check if this node represents a direct cast using unsafe
	src, dst, ok := detectUnsafeCast(expr)
	if !ok {
		return false
	}

	// it is a cast. Get the source and destination types
	srcType := getObjectType(src, pass)
	dstType := getObjectType(dst, pass)

	// then, check if the types are structs that contain a different amount of architecture-dependent types
	return checkIncompatibleStructsCast(srcType, dstType)
}

/*
 * checks if an AST node is a cast using unsafe
 */
func detectUnsafeCast(expr ast.Expr) (ast.Expr, ast.Expr, bool) {
	// find the potential cast expression: it might be a star expression (pointer), then take the inner node. Otherwise
	// we have the cast node directly
	var castExpr ast.Expr
	starExpr, ok := expr.(*ast.StarExpr)
	if ok {
		castExpr = starExpr.X
	} else {
		castExpr = expr
	}

	// check if the expression is a CallExpr, which is needed for it to be a cast
	targetCastCallExpr, ok := castExpr.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}

	// check if there are arguments to the call expression at all
	if len(targetCastCallExpr.Args) == 0 {
		return nil, nil, false
	}

	// now, extract the source expression in the cast and check if it is a CallExpr too. This is necessary because we
	// try to filter casts that involve an intermediate unsafe.Pointer step
	sourceCastCallExpr, ok := targetCastCallExpr.Args[0].(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}

	// extract the potential intermediate unsafe.Pointer step and check whether is is a selector expression
	sourceCastUnsafePointerSelector, ok := sourceCastCallExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}

	// now, check whether it is unsafe.Pointer indeed
	sourceCastUnsafePointerSelectorX, ok := sourceCastUnsafePointerSelector.X.(*ast.Ident)

	if sourceCastUnsafePointerSelector.Sel.Name != "Pointer" || sourceCastUnsafePointerSelectorX.Name != "unsafe" {
		return nil, nil, false
	}

	// we have an unsafe cast. Now, extract the source expression, and take care of a potential & operator
	var sourceExpr ast.Expr
	sourceUnary, ok := sourceCastCallExpr.Args[0].(*ast.UnaryExpr)
	if ok && sourceUnary.Op == token.AND {
		sourceExpr = sourceUnary.X
	} else {
		sourceExpr = sourceCastCallExpr.Args[0]
	}

	// extract the target expression, and take care of parenthesis and a star operator
	targetParen, ok := targetCastCallExpr.Fun.(*ast.ParenExpr)
	if !ok {
		return nil, nil, false
	}

	targetStar, ok := targetParen.X.(*ast.StarExpr)
	if !ok {
		return nil, nil, false
	}

	// finally, we know that this is a cast involving unsafe, return the source and destination
	return sourceExpr, targetStar.X, true
}

/**
 * finds the type of an expression from the type information contained in the analysis pass
 */
func getObjectType(expr ast.Expr, pass *analysis.Pass) types.Type {
	return pass.TypesInfo.Types[expr].Type.Underlying()
}

/**
 * for two types, checks whether they are incompatible, platform-dependent types
 */
func checkIncompatibleStructsCast(src types.Type, dst types.Type) bool {
	// check if the source type is a struct
	srcStruct, ok := src.(*types.Struct)
	if !ok {
		return false
	}

	// check if the destination type is a struct
	dstStruct, ok := dst.(*types.Struct)
	if !ok {
		return false
	}

	srcPlatformDependentCount := 0
	dstPlatformDependentCount := 0

	// count platform dependent types in the source type
	for i := 0; i < srcStruct.NumFields(); i++ {
		if isPlatformDependent(srcStruct.Field(i)) {
			srcPlatformDependentCount += 1
		}
	}

	// count platform dependent types in the destination type
	for i := 0; i < dstStruct.NumFields(); i++ {
		if isPlatformDependent(dstStruct.Field(i)) {
			dstPlatformDependentCount += 1
		}
	}

	// check whether the amounts match
	return srcPlatformDependentCount != dstPlatformDependentCount
}

/**
 * checks whether a type is platform dependent
 */
func isPlatformDependent(field *types.Var) bool {
	// check whether it is a basic type
	basicType, ok := field.Type().(*types.Basic)
	if !ok {
		return false
	}

	// check whether the basic type is int, uint, or uintptr
	kind := basicType.Kind()

	if kind == types.Int || kind == types.Uint || kind == types.Uintptr {
		return true
	}

	return false
}