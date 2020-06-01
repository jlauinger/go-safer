package sliceheader

import (
	"fmt"
	"golang.org/x/tools/go/cfg"

	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name: "sliceheader",
	Doc:  "reports reflect.SliceHeader and reflect.StringHeader misuses",
	Run:  run,
	Requires: []*analysis.Analyzer{inspect.Analyzer, ctrlflow.Analyzer},
	RunDespiteErrors: true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	fmt.Printf("") // to "need" fmt Package

	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	cfgResult := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	inspectResult.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		node := n.(*ast.CompositeLit)
		if compositeLiteralIsReflectHeader(node, pass) {
			pass.Reportf(n.Pos(), "reflect header composite literal found")
		}
		return true
	})

	inspectResult.WithStack([]ast.Node{(*ast.AssignStmt)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		node := n.(*ast.AssignStmt)
		if assigningToReflectHeader(node, pass, stack, cfgResult) {
			pass.Reportf(n.Pos(), "assigning to reflect header object")
		}
		return true
	})

	return nil, nil
}

func typeIsReflectHeader(t types.Type) bool {
	if t.String() == "invalid type" {
		return false
	}

	sliceHeaderType := types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "Data", types.Typ[types.Uintptr]),
		types.NewVar(token.NoPos, nil, "Len", types.Typ[types.Int]),
		types.NewVar(token.NoPos, nil, "Cap", types.Typ[types.Int]),
	}, nil)
	stringHeaderType := types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "Data", types.Typ[types.Uintptr]),
		types.NewVar(token.NoPos, nil, "Len", types.Typ[types.Int]),
	}, nil)

	var effectiveType types.Type
	pt, ok := t.Underlying().(*types.Pointer)
	if ok {
		effectiveType = pt.Elem().Underlying()
	} else {
		effectiveType = t.Underlying()
	}

	return types.AssignableTo(sliceHeaderType, effectiveType) || types.AssignableTo(stringHeaderType, effectiveType)
}

func typeIsSliceOrStringReferenceType(t types.Type) bool {
	stringType := types.NewPointer(types.Typ[types.String])
	if types.AssignableTo(stringType, t) {
		return true
	}

	if t.String()[0:3] == "*[]" {
		return true
	}

	return false
}

func compositeLiteralIsReflectHeader(cl *ast.CompositeLit, pass *analysis.Pass) bool {
	literalType, ok := pass.TypesInfo.Types[cl]
	if !ok {
		return false
	}

	return typeIsReflectHeader(literalType.Type)
}

func assigningToReflectHeader(assignStmt *ast.AssignStmt, pass *analysis.Pass, stack []ast.Node, cfgs *ctrlflow.CFGs) bool {
	var function *ast.FuncDecl
	for i := len(stack) - 1; i >= 0; i-- {
		fDecl, ok := stack[i].(*ast.FuncDecl)
		if ok {
			function = fDecl
			break
		}
	}

	for _, expr := range assignStmt.Lhs {
		lhs, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}

		lhsIdent, ok := lhs.X.(*ast.Ident)
		if !ok {
			lhsType := pass.TypesInfo.Types[lhs.X]
			return typeIsReflectHeader(lhsType.Type)
		}

		lhsObject := pass.TypesInfo.ObjectOf(lhsIdent)
		if lhsObject == nil {
			return true
		}


		if typeIsReflectHeader(lhsObject.Type()) {
			cfgStack := findPathInCFG(cfgs.FuncDecl(function), assignStmt)
			if derivedByCast(lhsObject, cfgStack, pass) {
				return false
			}
			return true
		}
	}
	return false
}

func findPathInCFG(cfg *cfg.CFG, stmt ast.Node) []ast.Node {
	_, stack := findPathInCFGIter(cfg.Blocks[0], &stmt, 0)
	return stack
}

func findPathInCFGIter(block *cfg.Block, stmt *ast.Node, depth int) (bool, []ast.Node) {
	if depth > 1000 { // avoid stack exhaustion
		return false, []ast.Node{}
	}

	contained, nodes := nodesUntilStmt(&block.Nodes, stmt)
	if contained {
		return true, nodes
	} else {
		for _, child := range block.Succs {
			found, childStack := findPathInCFGIter(child, stmt, depth + 1)
			if found {
				return true, append(block.Nodes, childStack...)
			}
		}
	}
	return false, []ast.Node{}
}

func nodesUntilStmt(nodes *[]ast.Node, stmt *ast.Node) (bool, []ast.Node) {
	count := 0
	for _, n := range *nodes {
		count++
		if n == *stmt {
			return true, (*nodes)[:count]
		}
	}
	return false, []ast.Node{}
}

func derivedByCast(object types.Object, cfgStack []ast.Node, pass *analysis.Pass) bool {
	var definitionExpr ast.Expr
	for i := len(cfgStack) - 1; i >= 0; i-- {
		node := cfgStack[i]
		assigns, valueExpr := nodeAssignsObject(node, object, pass)
		if assigns {
			definitionExpr = valueExpr
			break
		}
	}

	if definitionExpr == nil {
		return false
	}

	return definitionExprIsCastFromRealSlice(definitionExpr, pass)
}

func nodeAssignsObject(n ast.Node, object types.Object, pass *analysis.Pass) (bool, ast.Expr) {
	assignStmt, ok := n.(*ast.AssignStmt)
	if !ok {
		return false, nil
	}

	for i, lhs := range assignStmt.Lhs {
		lhsIdent, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		lhsObject, ok := pass.TypesInfo.Defs[lhsIdent]
		if !ok || lhsObject == nil {
			continue
		}
		if lhsObject.Id() == object.Id() {
			return true, assignStmt.Rhs[i]
		}
	}

	return false, nil
}

func definitionExprIsCastFromRealSlice(expr ast.Expr, pass *analysis.Pass) bool {
	var ok bool

	var callExpr *ast.CallExpr
	callExpr, ok = expr.(*ast.CallExpr)
	if !ok {
		starExpr, ok := expr.(*ast.StarExpr)
		if !ok {
			return false
		}
		callExpr, ok = starExpr.X.(*ast.CallExpr)
		if !ok {
			return false
		}
	}

	castTarget, ok := callExpr.Fun.(*ast.ParenExpr)
	if !ok {
		return false
	}

	castStarTarget, ok := castTarget.X.(*ast.StarExpr)
	if !ok {
		return false
	}

	castPointerTarget, ok := castStarTarget.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if castPointerTarget.Sel.Name != "SliceHeader" && castPointerTarget.Sel.Name != "StringHeader" {
		return false
	}

	selectorReceiver, ok := castPointerTarget.X.(*ast.Ident)
	if !ok {
		return false
	}

	if selectorReceiver.Name != "reflect" {
		return false
	}

	sourceCast, ok := callExpr.Args[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	sourceSelector, ok := sourceCast.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	sourceReceiver, ok := sourceSelector.X.(*ast.Ident)
	if !ok {
		return false
	}

	if sourceSelector.Sel.Name != "Pointer" || sourceReceiver.Name != "unsafe" {
		return false
	}

	sourceType, ok := pass.TypesInfo.Types[sourceCast.Args[0]]
	if !ok {
		return false
	}

	return typeIsSliceOrStringReferenceType(sourceType.Type)
}