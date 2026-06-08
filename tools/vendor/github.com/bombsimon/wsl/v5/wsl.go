package wsl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const (
	messageMissingWhitespaceAbove = "missing whitespace above this line"
	messageMissingWhitespaceBelow = "missing whitespace below this line"
	messageRemoveWhitespace       = "unnecessary whitespace"
)

type fixRange struct {
	fixRangeStart token.Pos
	fixRangeEnd   token.Pos
	fix           []byte
}

type issue struct {
	message string
	// We can report multiple fixes at the same position. This happens e.g. when
	// we force error cuddling but the error assignment is already cuddled.
	// See `checkError` for examples.
	fixRanges []fixRange
}

type WSL struct {
	file         *ast.File
	fset         *token.FileSet
	typeInfo     *types.Info
	issues       map[token.Pos]issue
	config       *Configuration
	groupedDecls map[token.Pos]struct{}
}

func New(file *ast.File, pass *analysis.Pass, cfg *Configuration) *WSL {
	return &WSL{
		fset:         pass.Fset,
		file:         file,
		typeInfo:     pass.TypesInfo,
		issues:       make(map[token.Pos]issue),
		config:       cfg,
		groupedDecls: make(map[token.Pos]struct{}),
	}
}

// Run will run analysis on the file and pass passed to the constructor. It's
// typically only supposed to be used by [analysis.Analyzer].
func (w *WSL) Run() {
	for _, decl := range w.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			w.checkFunc(funcDecl)
		}
	}
}

func (w *WSL) checkStmt(stmt ast.Stmt, cursor *Cursor) {
	//nolint:gocritic // This is not commented out code, it's examples
	switch s := stmt.(type) {
	// if a {} else if b {} else {}
	case *ast.IfStmt:
		w.checkIf(s, cursor, false)
	// for {} / for a; b; c {}
	case *ast.ForStmt:
		w.checkFor(s, cursor)
	// for _, _ = range a {}
	case *ast.RangeStmt:
		w.checkRange(s, cursor)
	// switch {} // switch a {}
	case *ast.SwitchStmt:
		w.checkSwitch(s, cursor)
	// switch a.(type) {}
	case *ast.TypeSwitchStmt:
		w.checkTypeSwitch(s, cursor)
	// return a
	case *ast.ReturnStmt:
		w.checkReturn(s, cursor)
	// continue / break
	case *ast.BranchStmt:
		w.checkBranch(s, cursor)
	// var a
	case *ast.DeclStmt:
		w.checkDeclStmt(s, cursor)
	// a := a
	case *ast.AssignStmt:
		w.checkAssign(s, cursor)
	// a++ / a--
	case *ast.IncDecStmt:
		w.checkIncDec(s, cursor)
	// defer func() {}
	case *ast.DeferStmt:
		w.checkDefer(s, cursor)
	// go func() {}
	case *ast.GoStmt:
		w.checkGo(s, cursor)
	// e.g. someFn()
	case *ast.ExprStmt:
		w.checkExprStmt(s, cursor)
	// case:
	case *ast.CaseClause:
		w.checkCaseClause(s, cursor)
	// case:
	case *ast.CommClause:
		w.checkCommClause(s, cursor)
	// { }
	case *ast.BlockStmt:
		w.checkBlock(s)
	// select { }
	case *ast.SelectStmt:
		w.checkSelect(s, cursor)
	// ch <- ...
	case *ast.SendStmt:
		w.checkSend(s, cursor)
	// LABEL:
	case *ast.LabeledStmt:
		w.checkLabel(s, cursor)
	case *ast.EmptyStmt:
	default:
	}
}

//nolint:unparam // False positive on `cursor`
func (w *WSL) checkExpr(expr ast.Expr, cursor *Cursor) {
	// This switch traverses all possible subexpressions in search
	// of anonymous functions, no matter how unlikely or perhaps even
	// semantically impossible it is.
	switch s := expr.(type) {
	case *ast.FuncLit:
		w.checkBlock(s.Body)
	case *ast.CallExpr:
		w.checkExpr(s.Fun, cursor)

		for _, e := range s.Args {
			w.checkExpr(e, cursor)
		}
	case *ast.StarExpr:
		w.checkExpr(s.X, cursor)
	case *ast.CompositeLit:
		w.checkExpr(s.Type, cursor)

		for _, e := range s.Elts {
			w.checkExpr(e, cursor)
		}
	case *ast.KeyValueExpr:
		w.checkExpr(s.Key, cursor)
		w.checkExpr(s.Value, cursor)
	case *ast.ArrayType:
		w.checkExpr(s.Elt, cursor)
		w.checkExpr(s.Len, cursor)
	case *ast.BasicLit:
	case *ast.BinaryExpr:
		w.checkExpr(s.X, cursor)
		w.checkExpr(s.Y, cursor)
	case *ast.ChanType:
		w.checkExpr(s.Value, cursor)
	case *ast.Ellipsis:
		w.checkExpr(s.Elt, cursor)
	case *ast.FuncType:
		if params := s.TypeParams; params != nil {
			for _, f := range params.List {
				w.checkExpr(f.Type, cursor)
			}
		}

		if params := s.Params; params != nil {
			for _, f := range params.List {
				w.checkExpr(f.Type, cursor)
			}
		}

		if results := s.Results; results != nil {
			for _, f := range results.List {
				w.checkExpr(f.Type, cursor)
			}
		}
	case *ast.Ident:
	case *ast.IndexExpr:
		w.checkExpr(s.Index, cursor)
		w.checkExpr(s.X, cursor)
	case *ast.IndexListExpr:
		w.checkExpr(s.X, cursor)

		for _, e := range s.Indices {
			w.checkExpr(e, cursor)
		}
	case *ast.InterfaceType:
		for _, f := range s.Methods.List {
			w.checkExpr(f.Type, cursor)
		}
	case *ast.MapType:
		w.checkExpr(s.Key, cursor)
		w.checkExpr(s.Value, cursor)
	case *ast.ParenExpr:
		w.checkExpr(s.X, cursor)
	case *ast.SelectorExpr:
		w.checkExpr(s.X, cursor)
	case *ast.SliceExpr:
		w.checkExpr(s.X, cursor)
		w.checkExpr(s.Low, cursor)
		w.checkExpr(s.High, cursor)
		w.checkExpr(s.Max, cursor)
	case *ast.StructType:
		for _, f := range s.Fields.List {
			w.checkExpr(f.Type, cursor)
		}
	case *ast.TypeAssertExpr:
		w.checkExpr(s.X, cursor)
		w.checkExpr(s.Type, cursor)
	case *ast.UnaryExpr:
		w.checkExpr(s.X, cursor)
	case nil:
	default:
	}
}

func (w *WSL) checkDecl(decl ast.Decl, cursor *Cursor) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			w.checkSpec(spec, cursor)
		}
	case *ast.FuncDecl:
		w.checkStmt(d.Body, cursor)
	case *ast.BadDecl:
	default:
	}
}

func (w *WSL) checkSpec(spec ast.Spec, cursor *Cursor) {
	switch s := spec.(type) {
	case *ast.ValueSpec:
		for _, expr := range s.Values {
			w.checkExpr(expr, cursor)
		}
	case *ast.ImportSpec, *ast.TypeSpec:
	default:
	}
}

func (w *WSL) checkBody(body []ast.Stmt) {
	cursor := NewCursor(body)

	for cursor.Next() {
		w.checkStmt(cursor.Stmt(), cursor)
	}
}

func (w *WSL) checkCuddlingBlock(
	stmt ast.Node,
	blockList []ast.Stmt,
	allowedIdents []*ast.Ident,
	cursor *Cursor,
	maxAllowedStatements int,
) {
	var firstBlockStmt ast.Node
	if len(blockList) > 0 {
		firstBlockStmt = blockList[0]
	}

	w.checkCuddlingMaxAllowed(stmt, firstBlockStmt, allowedIdents, cursor, maxAllowedStatements)
}

func (w *WSL) checkCuddling(stmt ast.Node, cursor *Cursor, maxAllowedStatements int) {
	w.checkCuddlingMaxAllowed(stmt, nil, []*ast.Ident{}, cursor, maxAllowedStatements)
}

func (w *WSL) checkCuddlingMaxAllowed(
	stmt ast.Node,
	firstBlockStmt ast.Node,
	allowedIdents []*ast.Ident,
	cursor *Cursor,
	maxAllowedStatements int,
) {
	if _, ok := cursor.Stmt().(*ast.LabeledStmt); ok {
		return
	}

	previousNode := cursor.PreviousNode()

	if previousNode != nil {
		if _, ok := w.groupedDecls[previousNode.End()]; ok {
			w.addErrorTooManyStatements(cursor.Stmt().Pos(), cursor.checkType)
			return
		}
	}

	numStmtsAbove := w.numberOfStatementsAbove(cursor)
	previousIdents := w.identsFromNode(previousNode, true)

	// If we don't have any statements above, we only care about potential error
	// cuddling (for if statements) so check that.
	if numStmtsAbove == 0 {
		w.checkError(numStmtsAbove, stmt, previousNode, cursor)
		return
	}

	nodeIsAssignDeclOrIncDec := func(n ast.Node) bool {
		_, a := n.(*ast.AssignStmt)
		_, d := n.(*ast.DeclStmt)
		_, i := n.(*ast.IncDecStmt)

		return a || d || i
	}

	_, currIsDefer := stmt.(*ast.DeferStmt)

	// We're cuddled but not with an assign, declare or defer statement which is
	// never allowed.
	if !nodeIsAssignDeclOrIncDec(previousNode) && !currIsDefer {
		w.addErrorInvalidTypeCuddle(cursor.Stmt().Pos(), cursor.checkType)
		return
	}

	checkIntersection := func(other []*ast.Ident) bool {
		anyIntersects := identIntersection(previousIdents, other)
		if len(anyIntersects) > 0 {
			// We have matches, but too many statements above.
			if maxAllowedStatements != -1 && numStmtsAbove > maxAllowedStatements {
				w.addErrorTooManyStatements(previousNode.Pos(), cursor.checkType)
			}

			return true
		}

		return false
	}

	// FEATURE(AllowWholeBlock): Allow identifier used anywhere in block
	// (including recursive blocks).
	if w.config.AllowWholeBlock {
		allIdentsInBlock := w.identsFromNode(stmt, false)
		if checkIntersection(allIdentsInBlock) {
			return
		}
	}

	// FEATURE(AllowFirstInBlock): Allow identifiers used first in block.
	if !w.config.AllowWholeBlock && w.config.AllowFirstInBlock {
		firstStmtIdents := w.identsFromNode(firstBlockStmt, true)
		if checkIntersection(firstStmtIdents) {
			return
		}
	}

	currentIdents := w.identsFromNode(stmt, true)
	if checkIntersection(currentIdents) {
		return
	}

	if checkIntersection(allowedIdents) {
		return
	}

	intersects := identIntersection(currentIdents, previousIdents)
	if len(intersects) > 0 {
		return
	}

	// We're cuddled but the line immediately above doesn't contain any
	// variables used in this statement.
	w.addErrorNoIntersection(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkCuddlingWithoutIntersection(stmt ast.Node, cursor *Cursor) {
	if w.numberOfStatementsAbove(cursor) == 0 {
		return
	}

	if _, ok := cursor.Stmt().(*ast.LabeledStmt); ok {
		return
	}

	previousNode := cursor.PreviousNode()

	currAssign, currIsAssign := stmt.(*ast.AssignStmt)
	previousAssign, prevIsAssign := previousNode.(*ast.AssignStmt)
	_, prevIsDecl := previousNode.(*ast.DeclStmt)
	_, prevIsIncDec := previousNode.(*ast.IncDecStmt)

	// Cuddling without intersection is allowed for assignments and inc/dec
	// statements. If however the check for declarations is disabled, we also
	// allow cuddling with them as well.
	//
	// var x string
	// x := ""
	// y++
	if _, ok := w.config.Checks[CheckDecl]; ok {
		prevIsDecl = false
	}

	// If we enable exclusive assign checks we only allow new declarations or
	// new assignments together but not mix and match.
	//
	// When this is enabled we also implicitly disable support to cuddle with
	// anything else.
	if _, ok := w.config.Checks[CheckAssignExclusive]; ok {
		prevIsDecl = false
		prevIsIncDec = false

		if prevIsAssign && currIsAssign {
			prevIsAssign = previousAssign.Tok == currAssign.Tok
		}
	}

	prevIsValidType := previousNode == nil || prevIsAssign || prevIsDecl || prevIsIncDec

	if _, ok := w.config.Checks[CheckAssignExpr]; !ok {
		if _, ok := previousNode.(*ast.ExprStmt); ok && w.hasIntersection(stmt, previousNode) {
			prevIsValidType = prevIsValidType || ok
		}
	}

	if prevIsValidType {
		return
	}

	w.addErrorInvalidTypeCuddle(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkBlock(block *ast.BlockStmt) {
	w.checkBlockLeadingNewline(block)
	w.checkTrailingNewline(block)

	w.checkBody(block.List)
}

func (w *WSL) checkCaseClause(stmt *ast.CaseClause, cursor *Cursor) {
	w.checkCaseLeadingNewline(stmt)

	if w.config.CaseMaxLines != 0 {
		w.checkCaseTrailingNewline(stmt.Body, cursor)
	}

	w.checkBody(stmt.Body)
}

func (w *WSL) checkCommClause(stmt *ast.CommClause, cursor *Cursor) {
	w.checkCommLeadingNewline(stmt)

	if w.config.CaseMaxLines != 0 {
		w.checkCaseTrailingNewline(stmt.Body, cursor)
	}

	w.checkBody(stmt.Body)
}

func (w *WSL) checkFunc(funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil {
		return
	}

	w.checkBlock(funcDecl.Body)
}

func (w *WSL) checkAssign(stmt *ast.AssignStmt, cursor *Cursor) {
	defer func() {
		for _, expr := range stmt.Rhs {
			w.checkExpr(expr, cursor)
		}

		w.checkAppend(stmt, cursor)
	}()

	if _, ok := w.config.Checks[CheckAssign]; !ok {
		return
	}

	cursor.SetChecker(CheckAssign)

	w.checkCuddlingWithoutIntersection(stmt, cursor)
}

func (w *WSL) checkAppend(stmt *ast.AssignStmt, cursor *Cursor) {
	if _, ok := w.config.Checks[CheckAppend]; !ok {
		return
	}

	if w.numberOfStatementsAbove(cursor) == 0 {
		return
	}

	previousNode := cursor.PreviousNode()

	var appendNode *ast.CallExpr

	for _, expr := range stmt.Rhs {
		e, ok := expr.(*ast.CallExpr)
		if !ok {
			continue
		}

		if f, ok := e.Fun.(*ast.Ident); ok && f.Name == "append" {
			appendNode = e
			break
		}
	}

	if appendNode == nil {
		return
	}

	if !w.hasIntersection(appendNode, previousNode) {
		w.addErrorNoIntersection(stmt.Pos(), CheckAppend)
	}
}

func (w *WSL) checkBranch(stmt *ast.BranchStmt, cursor *Cursor) {
	if _, ok := w.config.Checks[CheckBranch]; !ok {
		return
	}

	cursor.SetChecker(CheckBranch)

	if w.numberOfStatementsAbove(cursor) == 0 {
		return
	}

	lastStmtInBlock := cursor.statements[len(cursor.statements)-1]
	firstStmts := cursor.Nth(0)

	if w.lineFor(lastStmtInBlock.End())-w.lineFor(firstStmts.Pos()) < w.config.BranchMaxLines {
		return
	}

	w.addErrorTooManyLines(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkDeclStmt(stmt *ast.DeclStmt, cursor *Cursor) {
	w.checkDecl(stmt.Decl, cursor)

	if _, ok := w.config.Checks[CheckDecl]; !ok {
		return
	}

	cursor.SetChecker(CheckDecl)

	if w.numberOfStatementsAbove(cursor) == 0 {
		return
	}

	// Try to do smart grouping and if we succeed return, otherwise do
	// line-by-line fixing.
	if w.maybeGroupDecl(stmt, cursor) {
		return
	}

	w.addErrorNeverAllow(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkDefer(stmt *ast.DeferStmt, cursor *Cursor) {
	w.maybeCheckExpr(
		stmt,
		stmt.Call,
		cursor,
		func(n ast.Node) (int, bool) {
			_, previousIsDefer := n.(*ast.DeferStmt)
			_, previousIsIf := n.(*ast.IfStmt)

			// We allow defer as a third node only if we have an if statement
			// between, e.g.
			//
			// 	f, err := os.Open(file)
			// 	if err != nil {
			// 	    return err
			// 	}
			// defer f.Close()
			if previousIsIf && w.numberOfStatementsAbove(cursor) >= 2 {
				defer cursor.Save()()

				cursor.Previous()
				cursor.Previous()

				if w.hasIntersection(cursor.Stmt(), stmt) {
					return 1, false
				}
			}

			// Only check cuddling if previous statement isn't also a defer.
			return 1, !previousIsDefer
		},
		CheckDefer,
	)
}

func (w *WSL) checkError(
	stmtsAbove int,
	ifStmt ast.Node,
	previousNode ast.Node,
	cursor *Cursor,
) {
	if _, ok := w.config.Checks[CheckErr]; !ok {
		return
	}

	if _, ok := cursor.Stmt().(*ast.LabeledStmt); ok {
		return
	}

	defer cursor.Save()()

	// It must be an if statement
	stmt, ok := ifStmt.(*ast.IfStmt)
	if !ok {
		return
	}

	// If we actually have statements above we can't possibly need to remove any
	// empty lines.
	if stmtsAbove > 0 {
		return
	}

	// If the error checking has an init condition (e.g. if err := f();) we
	// don't want to check cuddling since the error is now assigned on this row.
	if stmt.Init != nil {
		return
	}

	// The condition must be a binary expression (X OP Y)
	binaryExpr, ok := stmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return
	}

	// We must do not equal or equal comparison (!= or ==)
	if binaryExpr.Op != token.NEQ && binaryExpr.Op != token.EQL {
		return
	}

	xIdent, ok := binaryExpr.X.(*ast.Ident)
	if !ok {
		return
	}

	// X is not an error so it's not error checking
	if !w.implementsErr(xIdent) {
		return
	}

	yIdent, ok := binaryExpr.Y.(*ast.Ident)
	if !ok {
		return
	}

	// Y is not compared with `nil`
	if yIdent.Name != "nil" {
		return
	}

	previousIdents := []*ast.Ident{}

	if assign, ok := previousNode.(*ast.AssignStmt); ok {
		for _, lhs := range assign.Lhs {
			previousIdents = append(previousIdents, w.identsFromNode(lhs, true)...)
		}
	}

	if decl, ok := previousNode.(*ast.DeclStmt); ok {
		if genDecl, ok := decl.Decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					previousIdents = append(previousIdents, vs.Names...)
				}
			}
		}
	}

	// Ensure that the error checked on this line was assigned or declared in
	// the previous statement.
	if len(identIntersection([]*ast.Ident{xIdent}, previousIdents)) == 0 {
		return
	}

	cursor.SetChecker(CheckErr)

	previousNodeEnd := previousNode.End()

	comments := ast.NewCommentMap(w.fset, previousNode, w.file.Comments)
	for _, cg := range comments {
		for _, c := range cg {
			if c.Pos() < previousNodeEnd || c.End() > ifStmt.Pos() {
				continue
			}

			if c.End() > previousNodeEnd {
				// There's a comment between the error variable and the
				// if-statement, we can't do much about this. Most likely, the
				// comment has a meaning, but even if not we would end up with
				// something like
				//
				// err := fn()
				// // Some Comment
				// if err != nil {}
				//
				// Which just feels marginally better than leaving the space
				// anyway.
				if w.lineFor(c.End()) != w.lineFor(previousNodeEnd) {
					return
				}

				// If they are on the same line though, we can just extend where
				// the line ends.
				previousNodeEnd = c.End()
			}
		}
	}

	w.addError(previousNodeEnd+1, previousNodeEnd, ifStmt.Pos(), messageRemoveWhitespace, cursor.checkType)

	// If we add the error at the same position but with a different fix
	// range, only the fix range will be updated.
	//
	//   a := 1
	//   err := fn()
	//
	//   if err != nil {}
	//
	// Should become
	//
	//   a := 1
	//
	//   err := fn()
	//   if err != nil {}
	cursor.Previous()

	// We report this fix on the same pos as the previous diagnostic, but the
	// fix is different. The reason is to just stack more fixes for the same
	// diagnostic, the issue isn't present until the first fix so this message
	// will never be shown to the user.
	if w.numberOfStatementsAbove(cursor) > 0 {
		w.addError(previousNodeEnd+1, previousNode.Pos(), previousNode.Pos(), messageMissingWhitespaceAbove, cursor.checkType)
	}
}

func (w *WSL) checkExprStmt(stmt *ast.ExprStmt, cursor *Cursor) {
	w.maybeCheckExpr(
		stmt,
		stmt.X,
		cursor,
		func(n ast.Node) (int, bool) {
			_, ok := n.(*ast.ExprStmt)
			return -1, !ok
		},
		CheckExpr,
	)
}

func (w *WSL) checkFor(stmt *ast.ForStmt, cursor *Cursor) {
	w.maybeCheckBlock(stmt, stmt.Body, cursor, CheckFor)
}

func (w *WSL) checkGo(stmt *ast.GoStmt, cursor *Cursor) {
	w.maybeCheckExpr(
		stmt,
		stmt.Call,
		cursor,
		// We can cuddle any amount `go` statements so only check cuddling if
		// the previous one isn't a `go` call.
		func(n ast.Node) (int, bool) {
			_, ok := n.(*ast.GoStmt)
			return 1, !ok
		},
		CheckGo,
	)
}

func (w *WSL) checkIf(stmt *ast.IfStmt, cursor *Cursor, isElse bool) {
	// if
	w.checkBlock(stmt.Body)

	switch v := stmt.Else.(type) {
	// else-if
	case *ast.IfStmt:
		w.checkIf(v, cursor, true)

	// else
	case *ast.BlockStmt:
		w.checkBlock(v)
	}

	if _, ok := w.config.Checks[CheckIf]; !isElse && ok {
		cursor.SetChecker(CheckIf)
		w.checkCuddlingBlock(stmt, stmt.Body.List, []*ast.Ident{}, cursor, 1)
	} else if _, ok := w.config.Checks[CheckErr]; !isElse && ok {
		previousNode := cursor.PreviousNode()

		w.checkError(
			w.numberOfStatementsAbove(cursor),
			stmt,
			previousNode,
			cursor,
		)
	}
}

func (w *WSL) checkIncDec(stmt *ast.IncDecStmt, cursor *Cursor) {
	defer w.checkExpr(stmt.X, cursor)

	if _, ok := w.config.Checks[CheckIncDec]; !ok {
		return
	}

	cursor.SetChecker(CheckIncDec)

	w.checkCuddlingWithoutIntersection(stmt, cursor)
}

func (w *WSL) checkLabel(stmt *ast.LabeledStmt, cursor *Cursor) {
	// We check the statement last because the statement is the same node as the
	// label (it's a labeled statement). This means that we _first_ want to
	// check any violations of cuddling the label (never cuddle label) before we
	// actually check the inner statement.
	//
	// It's a subtle difference, but it makes the diagnostic make more sense.
	// We do this by deferring the statmenet check so it happens last no matter
	// if we have label checking enabled or not.
	defer w.checkStmt(stmt.Stmt, cursor)

	if _, ok := w.config.Checks[CheckLabel]; !ok {
		return
	}

	cursor.SetChecker(CheckLabel)

	if w.numberOfStatementsAbove(cursor) == 0 {
		return
	}

	w.addErrorNeverAllow(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkRange(stmt *ast.RangeStmt, cursor *Cursor) {
	w.maybeCheckBlock(stmt, stmt.Body, cursor, CheckRange)
}

func (w *WSL) checkReturn(stmt *ast.ReturnStmt, cursor *Cursor) {
	for _, expr := range stmt.Results {
		w.checkExpr(expr, cursor)
	}

	if _, ok := w.config.Checks[CheckReturn]; !ok {
		return
	}

	cursor.SetChecker(CheckReturn)

	// There's only a return statement.
	if cursor.Len() <= 1 {
		return
	}

	if w.numberOfStatementsAbove(cursor) == 0 {
		return
	}

	// If the distance between the first statement and the return statement is
	// less than `n` LOC we're allowed to cuddle.
	firstStmts := cursor.Nth(0)
	if w.lineFor(stmt.End())-w.lineFor(firstStmts.Pos()) < w.config.BranchMaxLines {
		return
	}

	w.addErrorTooManyLines(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkSelect(stmt *ast.SelectStmt, cursor *Cursor) {
	w.maybeCheckBlock(stmt, stmt.Body, cursor, CheckSelect)
}

func (w *WSL) checkSend(stmt *ast.SendStmt, cursor *Cursor) {
	defer w.checkExpr(stmt.Value, cursor)

	if _, ok := w.config.Checks[CheckSend]; !ok {
		return
	}

	cursor.SetChecker(CheckSend)

	var stmts []ast.Stmt

	ast.Inspect(stmt.Value, func(n ast.Node) bool {
		if b, ok := n.(*ast.BlockStmt); ok {
			stmts = b.List
			return false
		}

		return true
	})

	w.checkCuddlingBlock(stmt, stmts, []*ast.Ident{}, cursor, 1)
}

func (w *WSL) checkSwitch(stmt *ast.SwitchStmt, cursor *Cursor) {
	w.maybeCheckBlock(stmt, stmt.Body, cursor, CheckSwitch)
}

func (w *WSL) checkTypeSwitch(stmt *ast.TypeSwitchStmt, cursor *Cursor) {
	w.maybeCheckBlock(stmt, stmt.Body, cursor, CheckTypeSwitch)
}

func (w *WSL) checkCaseTrailingNewline(body []ast.Stmt, cursor *Cursor) {
	if len(body) == 0 {
		return
	}

	defer cursor.Save()()

	if !cursor.Next() {
		return
	}

	var nextCase ast.Node

	switch n := cursor.Stmt().(type) {
	case *ast.CaseClause:
		nextCase = n
	case *ast.CommClause:
		nextCase = n
	default:
		return
	}

	firstStmt := body[0]
	lastStmt := body[len(body)-1]
	totalLines := w.lineFor(lastStmt.End()) - w.lineFor(firstStmt.Pos()) + 1

	if totalLines < w.config.CaseMaxLines {
		return
	}

	// Next case is not immediately after the last statement so must be newline
	// already.
	if w.lineFor(nextCase.Pos()) > w.lineFor(lastStmt.End())+1 {
		return
	}

	w.addError(lastStmt.End(), nextCase.Pos(), nextCase.Pos(), messageMissingWhitespaceBelow, CheckCaseTrailingNewline)
}

func (w *WSL) checkBlockLeadingNewline(body *ast.BlockStmt) {
	comments := ast.NewCommentMap(w.fset, body, w.file.Comments)
	w.checkLeadingNewline(body.Lbrace, body.List, comments)
}

func (w *WSL) checkCaseLeadingNewline(caseClause *ast.CaseClause) {
	comments := ast.NewCommentMap(w.fset, caseClause, w.file.Comments)
	w.checkLeadingNewline(caseClause.Colon, caseClause.Body, comments)
}

func (w *WSL) checkCommLeadingNewline(commClause *ast.CommClause) {
	comments := ast.NewCommentMap(w.fset, commClause, w.file.Comments)
	w.checkLeadingNewline(commClause.Colon, commClause.Body, comments)
}

func (w *WSL) checkLeadingNewline(startPos token.Pos, body []ast.Stmt, comments ast.CommentMap) {
	if _, ok := w.config.Checks[CheckLeadingWhitespace]; !ok {
		return
	}

	// No statements in the block, let's leave it as is.
	if len(body) == 0 {
		return
	}

	openLine := w.lineFor(startPos)
	openingPos := startPos + 1
	firstStmt := body[0].Pos()

	for _, commentGroup := range comments {
		for _, comment := range commentGroup {
			// The comment starts after the current opening position (originally
			// the LBrace) and ends before the current first statement
			// (originally first body.List item).
			if comment.Pos() > openingPos && comment.End() < firstStmt {
				openingPosLine := w.lineFor(openingPos)
				commentStartLine := w.lineFor(comment.Pos())

				// If comment starts at the same line as the opening position it
				// should just extend the position for the fixer if needed.
				// func fn() { // This comment starts at the same line as LBrace
				switch {
				// The comment is on the same line as current opening position.
				// E.g. func fn() { // A comment
				case commentStartLine == openingPosLine:
					openingPos = comment.End()
				// Opening position is the same as `{` and the comment is
				// directly on the line after (no empty line)
				case openingPosLine == openLine &&
					commentStartLine == openLine+1:
					openingPos = comment.End()
				// The opening position has been updated, it's another comment.
				case openingPosLine != openLine:
					openingPos = comment.End()
				// The opening position is still { and the comment is not
				// directly above - it must be an empty line which shouldn't be
				// there.
				default:
					firstStmt = comment.Pos()
				}
			}
		}
	}

	openingPosLine := w.lineFor(openingPos)
	firstStmtLine := w.lineFor(firstStmt)

	if firstStmtLine > openingPosLine+1 {
		w.addError(openingPos+1, openingPos, firstStmt, messageRemoveWhitespace, CheckLeadingWhitespace)
	}
}

func (w *WSL) checkTrailingNewline(body *ast.BlockStmt) {
	if _, ok := w.config.Checks[CheckTrailingWhitespace]; !ok {
		return
	}

	// No statements in the block, let's leave it as is.
	if len(body.List) == 0 {
		return
	}

	lastStmt := body.List[len(body.List)-1]

	// We don't want to force removal of the empty line for the last case since
	// it can be use for consistency and readability.
	if _, ok := lastStmt.(*ast.CaseClause); ok {
		return
	}

	closingPos := body.Rbrace
	lastStmtOrComment := lastStmt.End()

	// Empty label statements needs positional adjustment. #92
	if l, ok := lastStmt.(*ast.LabeledStmt); ok {
		if _, ok := l.Stmt.(*ast.EmptyStmt); ok {
			lastStmtOrComment = lastStmt.Pos()
		}
	}

	comments := ast.NewCommentMap(w.fset, body, w.file.Comments)
	for _, commentGroup := range comments {
		for _, comment := range commentGroup {
			if comment.End() < closingPos && comment.Pos() > lastStmtOrComment {
				lastStmtOrComment = comment.End()
			}
		}
	}

	closingPosLine := w.lineFor(closingPos)
	lastStmtLine := w.lineFor(lastStmtOrComment)

	if closingPosLine > lastStmtLine+1 {
		w.addError(lastStmtOrComment+1, lastStmtOrComment, closingPos, messageRemoveWhitespace, CheckTrailingWhitespace)
	}
}

func (w *WSL) maybeGroupDecl(stmt *ast.DeclStmt, cursor *Cursor) bool {
	firstNode := asGenDeclWithValueSpecs(cursor.PreviousNode())
	if firstNode == nil {
		return false
	}

	currentNode := asGenDeclWithValueSpecs(stmt)
	if currentNode == nil {
		return false
	}

	// Both are not same type, e.g. `const` or `var`
	if firstNode.Tok != currentNode.Tok {
		return false
	}

	group := &ast.GenDecl{
		Tok:    firstNode.Tok,
		Lparen: 1,
		Specs:  firstNode.Specs,
	}

	group.Specs = append(group.Specs, currentNode.Specs...)

	reportNodes := []ast.Node{currentNode}
	lastNode := currentNode

	for {
		nextPeeked := cursor.NextNode()
		if nextPeeked == nil {
			break
		}

		if w.lineFor(lastNode.End()) < w.lineFor(nextPeeked.Pos())-1 {
			break
		}

		nextNode := asGenDeclWithValueSpecs(nextPeeked)
		if nextNode == nil {
			break
		}

		if nextNode.Tok != firstNode.Tok {
			break
		}

		cursor.Next()

		group.Specs = append(group.Specs, nextNode.Specs...)
		reportNodes = append(reportNodes, nextNode)
		lastNode = nextNode
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), group); err != nil {
		return false
	}

	// We add a diagnostic to every subsequent statement to properly represent
	// the violations. Duplicate fixes for the same range is fine.
	for _, n := range reportNodes {
		w.groupedDecls[n.End()] = struct{}{}

		w.addErrorWithMessageAndFix(
			n.Pos(),
			firstNode.Pos(),
			lastNode.End(),
			fmt.Sprintf("%s (never cuddle %s)", messageMissingWhitespaceAbove, CheckDecl),
			buf.Bytes(),
		)
	}

	return true
}

func (w *WSL) maybeCheckBlock(
	node ast.Node,
	blockStmt *ast.BlockStmt,
	cursor *Cursor,
	check CheckType,
) {
	w.checkBlock(blockStmt)

	if _, ok := w.config.Checks[check]; ok {
		cursor.SetChecker(check)

		var (
			blockList     []ast.Stmt
			allowedIdents []*ast.Ident
		)

		if check != CheckSwitch && check != CheckTypeSwitch && check != CheckSelect {
			blockList = blockStmt.List
		} else {
			allowedIdents = w.identsFromCaseArms(node)
		}

		w.checkCuddlingBlock(node, blockList, allowedIdents, cursor, 1)
	}
}

func (w *WSL) maybeCheckExpr(
	node ast.Node,
	expr ast.Expr,
	cursor *Cursor,
	predicate func(ast.Node) (int, bool),
	check CheckType,
) {
	w.checkExpr(expr, cursor)

	if _, ok := w.config.Checks[check]; ok {
		cursor.SetChecker(check)
		previousNode := cursor.PreviousNode()

		if n, ok := predicate(previousNode); ok {
			w.checkCuddling(node, cursor, n)
		}
	}
}

// numberOfStatementsAbove will find out how many lines above the cursor's
// current statement there is without any newlines between.
func (w *WSL) numberOfStatementsAbove(cursor *Cursor) int {
	defer cursor.Save()()

	statementsWithoutNewlines := 0
	currentStmtStartLine := w.lineFor(cursor.Stmt().Pos())

	for cursor.Previous() {
		previousStmtEndLine := w.lineFor(cursor.Stmt().End())
		if previousStmtEndLine != currentStmtStartLine-1 {
			break
		}

		currentStmtStartLine = w.lineFor(cursor.Stmt().Pos())
		statementsWithoutNewlines++
	}

	return statementsWithoutNewlines
}

func (w *WSL) lineFor(pos token.Pos) int {
	return w.fset.PositionFor(pos, false).Line
}

func (w *WSL) implementsErr(node *ast.Ident) bool {
	typeInfo := w.typeInfo.TypeOf(node)
	if typeInfo == nil {
		return false
	}

	errorType, ok := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	if !ok {
		return false
	}

	return types.Implements(typeInfo, errorType)
}

func (w *WSL) addErrorInvalidTypeCuddle(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (invalid statement above %s)", messageMissingWhitespaceAbove, ct)
	w.addErrorWithMessage(pos, pos, pos, reportMessage)
}

func (w *WSL) addErrorTooManyStatements(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (too many statements above %s)", messageMissingWhitespaceAbove, ct)
	w.addErrorWithMessage(pos, pos, pos, reportMessage)
}

func (w *WSL) addErrorNoIntersection(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (no shared variables above %s)", messageMissingWhitespaceAbove, ct)
	w.addErrorWithMessage(pos, pos, pos, reportMessage)
}

func (w *WSL) addErrorTooManyLines(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (too many lines above %s)", messageMissingWhitespaceAbove, ct)
	w.addErrorWithMessage(pos, pos, pos, reportMessage)
}

func (w *WSL) addErrorNeverAllow(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (never cuddle %s)", messageMissingWhitespaceAbove, ct)
	w.addErrorWithMessage(pos, pos, pos, reportMessage)
}

func (w *WSL) addError(report, start, end token.Pos, message string, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (%s)", message, ct)
	w.addErrorWithMessage(report, start, end, reportMessage)
}

func (w *WSL) addErrorWithMessage(report, start, end token.Pos, message string) {
	w.addErrorWithMessageAndFix(report, start, end, message, []byte("\n"))
}

func (w *WSL) addErrorWithMessageAndFix(report, start, end token.Pos, message string, fix []byte) {
	iss, ok := w.issues[report]
	if !ok {
		iss = issue{
			message:   message,
			fixRanges: []fixRange{},
		}
	}

	iss.fixRanges = append(iss.fixRanges, fixRange{
		fixRangeStart: start,
		fixRangeEnd:   end,
		fix:           fix,
	})

	w.issues[report] = iss
}

func asGenDeclWithValueSpecs(n ast.Node) *ast.GenDecl {
	decl, ok := n.(*ast.DeclStmt)
	if !ok {
		return nil
	}

	genDecl, ok := decl.Decl.(*ast.GenDecl)
	if !ok {
		return nil
	}

	for _, spec := range genDecl.Specs {
		// We only care about value specs and not type specs or import
		// specs. We will never see any import specs but type specs we just
		// separate with an empty line as usual.
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			return nil
		}

		// It's very hard to get comments right in the ast and with the current
		// way the ast package works we simply don't support grouping at all if
		// there are any comments related to the node.
		if valueSpec.Doc != nil || valueSpec.Comment != nil {
			return nil
		}
	}

	return genDecl
}

func (w *WSL) hasIntersection(a, b ast.Node) bool {
	return len(w.nodeIdentIntersection(a, b)) > 0
}

func (w *WSL) nodeIdentIntersection(a, b ast.Node) []*ast.Ident {
	aI := w.identsFromNode(a, true)
	bI := w.identsFromNode(b, true)

	return identIntersection(aI, bI)
}

func identIntersection(a, b []*ast.Ident) []*ast.Ident {
	intersects := []*ast.Ident{}

	for _, as := range a {
		for _, bs := range b {
			if as.Name == bs.Name {
				intersects = append(intersects, as)
			}
		}
	}

	return intersects
}

func isTypeOrPredeclConst(obj types.Object) bool {
	switch o := obj.(type) {
	case *types.TypeName:
		// Covers predeclared types ("string", "int", ...) and user types.
		return true
	case *types.Const:
		// true/false/iota are universe consts.
		return o.Parent() == types.Universe
	case *types.Nil:
		return true
	case *types.PkgName:
		// Skip package qualifiers like "fmt" in fmt.Println
		return true
	default:
		return false
	}
}

// identsFromNode returns all *ast.Ident in a node except:
//   - type names (types.TypeName)
//   - builtin constants from the universe (true, false, iota)
//   - nil (*types.Nil)
//   - package names (types.PkgName)
//   - the blank identifier "_"
func (w *WSL) identsFromNode(node ast.Node, skipBlock bool) []*ast.Ident {
	var (
		idents []*ast.Ident
		seen   = map[string]struct{}{}
	)

	if node == nil {
		return idents
	}

	addIdent := func(ident *ast.Ident) {
		if ident == nil {
			return
		}

		name := ident.Name
		if name == "" || name == "_" {
			return
		}

		if _, ok := seen[name]; ok {
			return
		}

		idents = append(idents, ident)
		seen[name] = struct{}{}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if skipBlock {
			if _, ok := n.(*ast.BlockStmt); ok {
				return false
			}
		}

		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		// Prefer Uses over Defs; fall back to Defs if not a use site.
		var typesObject types.Object
		if obj := w.typeInfo.Uses[ident]; obj != nil {
			typesObject = obj
		} else if obj := w.typeInfo.Defs[ident]; obj != nil {
			typesObject = obj
		}

		// Unresolved (could be a build-tag or syntax artifact). Keep it.
		if typesObject == nil {
			addIdent(ident)
			return true
		}

		if isTypeOrPredeclConst(typesObject) {
			return true
		}

		addIdent(ident)

		return true
	})

	return idents
}

func (w *WSL) identsFromCaseArms(node ast.Node) []*ast.Ident {
	var (
		idents []*ast.Ident
		nodes  []ast.Stmt
		seen   = map[string]struct{}{}

		addUnseen = func(node ast.Node) {
			for _, ident := range w.identsFromNode(node, true) {
				if _, ok := seen[ident.Name]; ok {
					continue
				}

				seen[ident.Name] = struct{}{}
				idents = append(idents, ident)
			}
		}
	)

	switch v := node.(type) {
	case *ast.SwitchStmt:
		nodes = v.Body.List
	case *ast.TypeSwitchStmt:
		nodes = v.Body.List
	case *ast.SelectStmt:
		nodes = v.Body.List
	default:
		return idents
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.CommClause:
			addUnseen(n.Comm)
		case *ast.CaseClause:
			for _, n := range n.List {
				addUnseen(n)
			}
		default:
			continue
		}
	}

	return idents
}
