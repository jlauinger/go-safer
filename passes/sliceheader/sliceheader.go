package sliceheader

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/cfg"
)

// Analyzer is a golang.org/x/tools/go/analysis style linter pass.
// Use this with the Vet-style infrastructure.
var Analyzer = &analysis.Analyzer{
	Name:             "sliceheader",
	Doc:              "reports reflect.SliceHeader and reflect.StringHeader misuses",
	Run:              run,
	Requires:         []*analysis.Analyzer{inspect.Analyzer, ctrlflow.Analyzer},
	RunDespiteErrors: true,
}

/**
 * run is the entry point to the analysis pass
 */
func run(pass *analysis.Pass) (interface{}, error) {
	// get results from required inspect and control flow graph analyzers
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	cfgResult := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	// filter AST of package under analysis for composite literal nodes, which are the first possible node to find a
	// slice header misuse
	inspectResult.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		node := n.(*ast.CompositeLit)
		// check if the node is a reflect header (slice or string) literal and report a warning if so
		if compositeLiteralIsReflectHeader(node, pass) {
			pass.Reportf(n.Pos(), "reflect header composite literal found")
		}
		return true
	})

	// filter AST of package under analysis for assignment statement nodes, which are the second possible
	inspectResult.WithStack([]ast.Node{(*ast.AssignStmt)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		node := n.(*ast.AssignStmt)
		// check if the assignment is done to a reflect header that was derived incorrectly and report a warning if so
		if assigningToReflectHeader(node, pass, stack, cfgResult) {
			pass.Reportf(n.Pos(), "assigning to incorrectly derived reflect header object")
		}
		return true
	})

	return nil, nil
}

/**
 * checks whether a type is a reflect header or similar type
 */
func typeIsReflectHeader(t types.Type) bool {
	// filter out possible parsing errors / invalid types
	if t.String() == "invalid type" {
		return false
	}

	// make up the basic structure of slice and string headers to compare to
	sliceHeaderType := types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "Data", types.Typ[types.Uintptr]),
		types.NewVar(token.NoPos, nil, "Len", types.Typ[types.Int]),
		types.NewVar(token.NoPos, nil, "Cap", types.Typ[types.Int]),
	}, nil)
	stringHeaderType := types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "Data", types.Typ[types.Uintptr]),
		types.NewVar(token.NoPos, nil, "Len", types.Typ[types.Int]),
	}, nil)

	// find the underlying type, taking care of a possible pointer type
	var effectiveType types.Type
	pt, ok := t.Underlying().(*types.Pointer)
	if ok {
		effectiveType = pt.Elem().Underlying()
	} else {
		effectiveType = t.Underlying()
	}

	// check if the types are the same by checking whether they are assignable
	return types.AssignableTo(sliceHeaderType, effectiveType) || types.AssignableTo(stringHeaderType, effectiveType)
}

/**
 * checks whether a type is a reference to a real slice or string
 */
func typeIsSliceOrStringReferenceType(t types.Type) bool {
	// check string type by creating a string type dummy and checking if the types are assignable
	stringType := types.NewPointer(types.Typ[types.String])
	if types.AssignableTo(stringType, t) {
		return true
	}

	// check slice type by comparing the string representation of the type: it is a slice if it starts with *[]
	if t.String()[0:3] == "*[]" {
		return true
	}

	// otherwise, it must be something else
	return false
}

/*
 * checks whether a composite literal AST node represents a literal of a reflect header type
 */
func compositeLiteralIsReflectHeader(cl *ast.CompositeLit, pass *analysis.Pass) bool {
	// get the type information for this node
	literalType, ok := pass.TypesInfo.Types[cl]
	if !ok {
		return false
	}

	// check if the type is a reflect header
	return typeIsReflectHeader(literalType.Type)
}

/**
 * checks if an assignment statement AST node is an assignment to a reflect header that is incorrectly derived
 */
func assigningToReflectHeader(assignStmt *ast.AssignStmt, pass *analysis.Pass, stack []ast.Node, cfgs *ctrlflow.CFGs) bool {
	// find the function that contains the assignment statement by looking up the parsing stack
	var function *ast.FuncDecl
	for i := len(stack) - 1; i >= 0; i-- {
		fDecl, ok := stack[i].(*ast.FuncDecl)
		if ok {
			function = fDecl
			break
		}
	}

	// an assignment statement can have multiple assignments, therefore analyze all of them
	for _, expr := range assignStmt.Lhs {
		// check if this assignment target is a selector expression, because only those can refer to fields in a
		// reflect header struct type
		lhs, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}

		// get the struct part of the assignment target and check that it is an identifier that we can analyze
		lhsIdent, ok := lhs.X.(*ast.Ident)
		if !ok {
			// if it isn't an identifier, get the type of it and check if it is a reflect header
			lhsType := pass.TypesInfo.Types[lhs.X]
			return typeIsReflectHeader(lhsType.Type)
		}

		// if it is an identifier, we can now check whether it was derived safely. First, dereference the identifier
		// to the object it refers to
		lhsObject := pass.TypesInfo.ObjectOf(lhsIdent)
		if lhsObject == nil {
			// if the object cannot be found, we assume it was not derived safely and therefore return true
			return true
		}

		// check if the object is a reflect header type
		if typeIsReflectHeader(lhsObject.Type()) {
			// now, find the path in the control flow graph that is used to construct the object
			cfgStack := findPathInCFG(cfgs.FuncDecl(function), assignStmt)
			// then check if it was derived by a safe cast from a real slice or string, and return true/false
			// accordingly
			if derivedByCast(lhsObject, cfgStack, pass) {
				return false
			}
			return true
		}
	}
	// in the default case, it is not a reflect header target and therefore not warned
	return false
}

/**
 * finds the path of nodes in a control flow graph up to a specified needle node
 */
func findPathInCFG(cfg *cfg.CFG, stmt ast.Node) []ast.Node {
	// use a recursive iterator function that keeps track of the depth, initialized to 0
	_, stack := findPathInCFGIter(cfg.Blocks[0], &stmt, 0)
	return stack
}

/**
 * iteratively finds a path from a starting CFG block to a node
 */
func findPathInCFGIter(block *cfg.Block, stmt *ast.Node, depth int) (bool, []ast.Node) {
	if depth > 1000 { // avoid stack exhaustion
		return false, []ast.Node{}
	}

	// check if the node is contained in this CFG block, if so we can return true and the nodes
	contained, nodes := nodesUntilStmt(&block.Nodes, stmt)
	if contained {
		return true, nodes
	}

	// otherwise, go through all children blocks and search the node there
	for _, child := range block.Succs {
		// search the node in this child, with incremented depth as a termination condition
		found, childStack := findPathInCFGIter(child, stmt, depth+1)
		if found {
			// if the node is found somewhere in this child, prepend the nodes in the current block to all nodes
			// on the path starting from the child, recursively returned
			return true, append(block.Nodes, childStack...)
		}
	}

	// otherwise, the node is not found
	return false, []ast.Node{}
}

/**
 * returns the nods in a CFG block up to a specified needle node, if it is contained in the block
 */
func nodesUntilStmt(nodes *[]ast.Node, stmt *ast.Node) (bool, []ast.Node) {
	count := 0
	// go through all nodes in the block
	for _, n := range *nodes {
		count++
		// if it is the node we are searching for, return a slice of the nodes up to this point
		if n == *stmt {
			return true, (*nodes)[:count]
		}
	}
	// if the node can't be found, return an empty nodes list
	return false, []ast.Node{}
}

/**
 * checks if an object is derived with a cast from a real slice or string
 */
func derivedByCast(object types.Object, cfgStack []ast.Node, pass *analysis.Pass) bool {
	// we need to find the expression that defines the object under analysis
	var definitionExpr ast.Expr
	// go through the stack of nodes in the CFG, starting from the back because we are interested in the last assignment
	// as this is defining the actual value of the object
	for i := len(cfgStack) - 1; i >= 0; i-- {
		node := cfgStack[i]
		// check if this CFG node is an assignment to the object
		assigns, valueExpr := nodeAssignsObject(node, object, pass)
		if assigns {
			definitionExpr = valueExpr
			break
		}
	}

	// if we cannot find an assignment to the object, we infer it was not derived safely by a cast
	if definitionExpr == nil {
		return false
	}

	// otherwise, check if the defining statement is a cast from a real slice or string
	return definitionExprIsCastFromRealSlice(definitionExpr, pass)
}

/**
 * checks if an AST node is an assignment of a given object
 */
func nodeAssignsObject(n ast.Node, object types.Object, pass *analysis.Pass) (bool, ast.Expr) {
	// check if the node is an assignment statement at all
	assignStmt, ok := n.(*ast.AssignStmt)
	if !ok {
		return false, nil
	}

	// if so, go through all the left hand side assignment targets of this node
	for i, lhs := range assignStmt.Lhs {
		// check that the assignment target is an object identifier
		lhsIdent, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		// try to find the target object
		lhsObject, ok := pass.TypesInfo.Defs[lhsIdent]
		if !ok || lhsObject == nil {
			continue
		}
		// check if the objects match and if so return the source expression, i.e. the value that is being assigned to
		// the object
		if lhsObject.Id() == object.Id() {
			return true, assignStmt.Rhs[i]
		}
	}

	// otherwise, this node is not an assignment to the object
	return false, nil
}

/**
 * checks whether an expression is a cast from a slice or string
 */
func definitionExprIsCastFromRealSlice(expr ast.Expr, pass *analysis.Pass) bool {
	var ok bool

	// check if the expression is a call expression
	var callExpr *ast.CallExpr
	callExpr, ok = expr.(*ast.CallExpr)
	if !ok {
		// if not, check if it is a call expression that is preceded by a star operator
		starExpr, ok := expr.(*ast.StarExpr)
		if !ok {
			return false
		}
		callExpr, ok = starExpr.X.(*ast.CallExpr)
		if !ok {
			return false
		}
	}

	// check that the cast target is a reflect header structure
	if !definitionCastPointerTargetIsReflectHeader(callExpr) {
		return false
	}

	// next, check that there is an intermediate cast through unsafe.Pointer by checking if the cast source is also a
	// call expression
	sourceCast, ok := callExpr.Args[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	// check that the callee is a selector expression (unsafe.Pointer is)
	sourceSelector, ok := sourceCast.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// convert the selector receiver to an identifier
	sourceReceiver, ok := sourceSelector.X.(*ast.Ident)
	if !ok {
		return false
	}

	// check that the intermediate cast is indeed to unsafe.Pointer
	if sourceSelector.Sel.Name != "Pointer" || sourceReceiver.Name != "unsafe" {
		return false
	}

	// try to find the type of the argument to the unsafe.Pointer cast, that is the original source
	sourceType, ok := pass.TypesInfo.Types[sourceCast.Args[0]]
	if !ok {
		return false
	}

	// check whether that source type is a string or slice reference
	return typeIsSliceOrStringReferenceType(sourceType.Type)
}

func definitionCastPointerTargetIsReflectHeader(callExpr *ast.CallExpr) bool {
	var ok bool

	// check that the cast callee is a parenthesis expression
	castTarget, ok := callExpr.Fun.(*ast.ParenExpr)
	if !ok {
		return false
	}

	// then check that inside the parenthesis there is a star expression, which is required because we will be casting
	// from unsafe.Pointer
	castStarTarget, ok := castTarget.X.(*ast.StarExpr)
	if !ok {
		return false
	}

	// the star must be in front of a selector expression (e.g. reflect.SliceHeader)
	castPointerTarget, ok := castStarTarget.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// first, check if the selector is either that of a slice or string header
	if castPointerTarget.Sel.Name != "SliceHeader" && castPointerTarget.Sel.Name != "StringHeader" {
		return false
	}

	// if so, convert the selector receiver to an identifier
	selectorReceiver, ok := castPointerTarget.X.(*ast.Ident)
	if !ok {
		return false
	}

	// and check if it is indeed reflect
	if selectorReceiver.Name != "reflect" {
		return false
	}

	// if all conditions hold, return true
	return true
}
