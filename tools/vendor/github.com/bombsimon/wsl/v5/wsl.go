package wsl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"slices"

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
	ast.Inspect(w.file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			w.checkBlock(node.Body, NewCursor([]ast.Stmt{}))
		case *ast.FuncLit:
			w.checkBlock(node.Body, NewCursor([]ast.Stmt{}))
		}

		return true
	})
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
		w.checkBlock(s, cursor)
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

	if w.isLockOrUnlock(stmt, previousNode) {
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

	if w.isLockOrUnlock(stmt, previousNode) {
		return
	}

	w.addErrorInvalidTypeCuddle(stmt.Pos(), cursor.checkType)
}

func (w *WSL) checkBlock(block *ast.BlockStmt, cursor *Cursor) {
	// Block can be nil for function declarations without a body.
	if block == nil {
		return
	}

	w.checkBlockLeadingNewline(block)
	w.checkTrailingNewline(block)
	w.checkNewlineAfterBlock(block, cursor)

	w.checkBody(block.List)
}

func (w *WSL) checkNewlineAfterBlock(block *ast.BlockStmt, cursor *Cursor) {
	if _, ok := w.config.Checks[CheckAfterBlock]; !ok {
		return
	}

	// For function blocks we don't have any statements in our cursor.
	if cursor.Len() == 0 {
		return
	}

	defer cursor.Save()()

	// Capture current statement and previous node before moving cursor.
	currentStmt := cursor.Stmt()
	previousNode := cursor.PreviousNode()

	if !cursor.Next() {
		// No more statements after this one so check for comments after.
		// Skip comments that are inside the current statement (e.g., inside an else block).
		if cPos := w.commentOnLineAfterNodePos(block); cPos != token.NoPos && cPos >= currentStmt.End() {
			insertPos := w.lineStartOf(cPos)
			w.addError(
				block.Rbrace,
				insertPos,
				insertPos,
				messageMissingWhitespaceBelow,
				CheckAfterBlock,
			)
		}

		return
	}

	// Exception: if err != nil { } followed by defer that references
	// a variable assigned above the if block.
	if w.isErrNotNilCheck(currentStmt) != nil {
		if deferStmt, ok := cursor.Stmt().(*ast.DeferStmt); ok && previousNode != nil {
			if w.hasIntersection(previousNode, deferStmt) {
				return
			}
		}
	}

	rBraceLine := w.lineFor(block.Rbrace)
	nextContentPos := cursor.Stmt().Pos()
	nextContentLine := w.lineFor(nextContentPos)

	// Find the first comment between rbrace and the next statement.
	for _, cg := range w.file.Comments {
		if cg.End() <= block.Rbrace {
			continue
		}

		// Skip comments that are inside the current statement but after this block.
		// This handles cases like comments inside an else block when checking the if-body.
		if cg.Pos() < currentStmt.End() {
			continue
		}

		if w.lineFor(cg.End()) == rBraceLine {
			continue
		}

		commentLine := w.lineFor(cg.Pos())
		if commentLine > rBraceLine && commentLine < nextContentLine {
			nextContentPos = cg.Pos()
			nextContentLine = commentLine
		}

		break
	}

	if nextContentLine <= rBraceLine+1 {
		insertPos := w.lineStartOf(nextContentPos)
		w.addError(
			block.Rbrace,
			insertPos,
			insertPos,
			messageMissingWhitespaceBelow,
			CheckAfterBlock,
		)
	}
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

func (w *WSL) checkAssign(stmt *ast.AssignStmt, cursor *Cursor) {
	defer w.checkAppend(stmt, cursor)

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

	if stmtsAbove > 0 {
		return
	}

	if _, ok := cursor.Stmt().(*ast.LabeledStmt); ok {
		return
	}

	defer cursor.Save()()

	errIdent := w.isErrNotNilCheck(ifStmt)
	if errIdent == nil {
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
	if len(identIntersection([]*ast.Ident{errIdent}, previousIdents)) == 0 {
		return
	}

	cursor.SetChecker(CheckErr)

	previousEndLine := w.lineFor(previousNode.End())

	// Check for comments on the same line as the previous node (extends effective end line).
	for _, cg := range w.file.Comments {
		if cg.Pos() >= ifStmt.Pos() {
			break
		}

		if cg.Pos() < previousNode.End() || cg.End() > ifStmt.Pos() {
			continue
		}

		// There's a comment between the error variable and the if-statement.
		// If it's on a different line, we can't do much about this.
		if w.lineFor(cg.End()) != previousEndLine {
			return
		}

		// Comment is on the same line - no need to update since line stays the same.
	}

	ifStmtLine := w.lineFor(ifStmt.Pos())
	file := w.fset.File(ifStmt.Pos())

	// Remove blank lines between previous node and if statement.
	removeStart := file.LineStart(previousEndLine + 1)
	removeEnd := file.LineStart(ifStmtLine)
	w.addErrorRemoveNewline(removeStart, removeEnd, cursor.checkType)

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

	// Add whitespace above the error assignment if there's a statement above.
	if w.numberOfStatementsAbove(cursor) > 0 {
		insertPos := w.lineStartOf(previousNode.Pos())
		w.addError(removeStart, insertPos, insertPos, messageMissingWhitespaceAbove, cursor.checkType)
	}
}

func (w *WSL) checkExprStmt(stmt *ast.ExprStmt, cursor *Cursor) {
	w.maybeCheckExpr(
		stmt,
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
	w.checkBlock(stmt.Body, cursor)

	switch v := stmt.Else.(type) {
	// else-if
	case *ast.IfStmt:
		w.checkIf(v, cursor, true)

	// else
	case *ast.BlockStmt:
		w.checkBlock(v, cursor)
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

	var (
		firstStmt  = body[0]
		lastStmt   = body[len(body)-1]
		totalLines = w.lineFor(nextCase.Pos()) - w.lineFor(firstStmt.Pos())
	)

	if totalLines < w.config.CaseMaxLines {
		return
	}

	var (
		lastStmtEndLine = w.lineFor(lastStmt.End())
		nextCaseLine    = w.lineFor(nextCase.Pos())
		nextCaseCol     = w.fset.PositionFor(nextCase.Pos(), false).Column
	)

	// Find transition point between trailing content (indented) and leading
	// content (left-aligned). Trailing comments belong to current case, leading
	// comments belong to next case. The blank line goes at the transition.
	var (
		lastStmtOrCommentEnd         = lastStmt.End()
		nextCaseOrLeftAlignedComment = nextCase.Pos()
		lastLeftAlignedCommentEnd    = token.NoPos
	)

	for _, commentGroup := range w.file.Comments {
		if commentGroup.Pos() >= nextCase.Pos() {
			break
		}

		if commentGroup.End() <= lastStmt.End() {
			continue
		}

		for _, comment := range commentGroup.List {
			commentLine := w.lineFor(comment.Pos())
			if commentLine <= lastStmtEndLine || commentLine >= nextCaseLine {
				continue
			}

			col := w.fset.PositionFor(comment.Pos(), false).Column
			if col <= nextCaseCol {
				// Left-aligned: first one marks transition point
				if lastLeftAlignedCommentEnd == token.NoPos {
					nextCaseOrLeftAlignedComment = comment.Pos()
				}

				lastLeftAlignedCommentEnd = comment.End()
			} else {
				// Indented: extend trailing content
				lastStmtOrCommentEnd = comment.End()
			}
		}
	}

	lastStmtOrCommentLine := w.lineFor(lastStmtOrCommentEnd)
	nextCaseOrLeadingCommentLine := w.lineFor(nextCaseOrLeftAlignedComment)

	// Check for unnecessary blank line before case (leading comments should be flush).
	if lastLeftAlignedCommentEnd != token.NoPos {
		lastLeadingEndLine := w.lineFor(lastLeftAlignedCommentEnd)

		if lastLeadingEndLine < nextCaseLine-1 {
			file := w.fset.File(nextCase.Pos())
			w.addErrorRemoveNewline(file.LineStart(lastLeadingEndLine+1), file.LineStart(nextCaseLine), CheckCaseTrailingNewline)
		}
	}

	// Already has a blank line at the boundary.
	if nextCaseOrLeadingCommentLine > lastStmtOrCommentLine+1 {
		return
	}

	insertPos := w.lineStartOf(nextCaseOrLeftAlignedComment)
	w.addError(lastStmtOrCommentEnd, insertPos, insertPos, messageMissingWhitespaceBelow, CheckCaseTrailingNewline)
}

func (w *WSL) checkBlockLeadingNewline(body *ast.BlockStmt) {
	w.checkLeadingNewline(body.Lbrace, body.List)
}

func (w *WSL) checkCaseLeadingNewline(caseClause *ast.CaseClause) {
	w.checkLeadingNewline(caseClause.Colon, caseClause.Body)
}

func (w *WSL) checkCommLeadingNewline(commClause *ast.CommClause) {
	w.checkLeadingNewline(commClause.Colon, commClause.Body)
}

func (w *WSL) checkLeadingNewline(startPos token.Pos, body []ast.Stmt) {
	if _, ok := w.config.Checks[CheckLeadingWhitespace]; !ok {
		return
	}

	if len(body) == 0 {
		return
	}

	var (
		openLine        = w.lineFor(startPos)
		firstStmtPos    = body[0].Pos()
		firstStmtLine   = w.lineFor(firstStmtPos)
		leadingComments []*ast.CommentGroup
	)

	for _, cg := range w.file.Comments {
		if cg.Pos() >= firstStmtPos {
			break
		}

		if cg.Pos() > startPos {
			leadingComments = append(leadingComments, cg)
		}
	}

	if len(leadingComments) == 0 {
		if firstStmtLine := w.lineFor(firstStmtPos); firstStmtLine > openLine+1 {
			file := w.fset.File(startPos)
			w.addErrorRemoveNewline(
				file.LineStart(openLine+1),
				file.LineStart(firstStmtLine),
				CheckLeadingWhitespace,
			)
		}

		return
	}

	var (
		firstContentLine   = firstStmtLine
		lastCommentEndLine = openLine
	)

	for _, comment := range leadingComments {
		startLine := w.lineFor(comment.Pos())
		endLine := w.lineFor(comment.End())

		if startLine > openLine && startLine < firstContentLine {
			firstContentLine = startLine
		}

		if endLine > lastCommentEndLine {
			lastCommentEndLine = endLine
		}
	}

	file := w.fset.File(startPos)

	// Empty line after opening brace.
	if firstContentLine > openLine+1 {
		w.addErrorRemoveNewline(
			file.LineStart(openLine+1),
			file.LineStart(firstContentLine),
			CheckLeadingWhitespace,
		)
	}

	// Empty line between comments and first statement.
	if lastCommentEndLine > openLine && firstStmtLine > lastCommentEndLine+1 {
		w.addErrorRemoveNewline(
			file.LineStart(lastCommentEndLine+1),
			file.LineStart(firstStmtLine),
			CheckLeadingWhitespace,
		)
	}
}

func (w *WSL) checkTrailingNewline(body *ast.BlockStmt) {
	if _, ok := w.config.Checks[CheckTrailingWhitespace]; !ok {
		return
	}

	if len(body.List) == 0 {
		return
	}

	lastStmt := body.List[len(body.List)-1]

	// We don't want to force removal of the empty line for the last case since
	// it can be used for consistency and readability.
	if _, ok := lastStmt.(*ast.CaseClause); ok {
		return
	}

	lastContentPos := lastStmt.End()

	// Empty label statements need positional adjustment. #92
	if l, ok := lastStmt.(*ast.LabeledStmt); ok {
		if _, ok := l.Stmt.(*ast.EmptyStmt); ok {
			lastContentPos = lastStmt.Pos()
		}
	}

	// Find the last comment after last statement using position comparison.
	for _, cg := range w.file.Comments {
		if cg.End() <= lastContentPos {
			continue
		}

		if cg.Pos() >= body.Rbrace {
			break
		}

		if cg.End() < body.Rbrace {
			lastContentPos = cg.End()
		}
	}

	closingLine := w.lineFor(body.Rbrace)
	lastContentLine := w.lineFor(lastContentPos)

	if closingLine > lastContentLine+1 {
		file := w.fset.File(body.Rbrace)
		removeStart := file.LineStart(lastContentLine + 1)
		removeEnd := file.LineStart(closingLine)
		w.addErrorRemoveNewline(removeStart, removeEnd, CheckTrailingWhitespace)
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
	w.checkBlock(blockStmt, cursor)

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
	cursor *Cursor,
	predicate func(ast.Node) (int, bool),
	check CheckType,
) {
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

func (w *WSL) lineStartOf(pos token.Pos) token.Pos {
	return w.fset.File(pos).LineStart(w.lineFor(pos))
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

func (w *WSL) commentOnLineAfterNodePos(node ast.Node) token.Pos {
	nodeEndLine := w.lineFor(node.End())

	for _, cg := range w.file.Comments {
		if cg.End() <= node.End() {
			continue
		}

		commentLine := w.lineFor(cg.Pos())
		if commentLine == nodeEndLine {
			continue
		}

		if commentLine == nodeEndLine+1 {
			return cg.Pos()
		}

		break
	}

	return token.NoPos
}

func (w *WSL) addErrorInvalidTypeCuddle(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (invalid statement above %s)", messageMissingWhitespaceAbove, ct)
	insertPos := w.lineStartOf(pos)
	w.addErrorWithMessage(pos, insertPos, insertPos, reportMessage)
}

func (w *WSL) addErrorTooManyStatements(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (too many statements above %s)", messageMissingWhitespaceAbove, ct)
	insertPos := w.lineStartOf(pos)
	w.addErrorWithMessage(pos, insertPos, insertPos, reportMessage)
}

func (w *WSL) addErrorNoIntersection(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (no shared variables above %s)", messageMissingWhitespaceAbove, ct)
	insertPos := w.lineStartOf(pos)
	w.addErrorWithMessage(pos, insertPos, insertPos, reportMessage)
}

func (w *WSL) addErrorTooManyLines(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (too many lines above %s)", messageMissingWhitespaceAbove, ct)
	insertPos := w.lineStartOf(pos)
	w.addErrorWithMessage(pos, insertPos, insertPos, reportMessage)
}

func (w *WSL) addErrorNeverAllow(pos token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (never cuddle %s)", messageMissingWhitespaceAbove, ct)
	insertPos := w.lineStartOf(pos)
	w.addErrorWithMessage(pos, insertPos, insertPos, reportMessage)
}

func (w *WSL) addError(report, start, end token.Pos, message string, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (%s)", message, ct)
	w.addErrorWithMessage(report, start, end, reportMessage)
}

func (w *WSL) addErrorRemoveNewline(start, end token.Pos, ct CheckType) {
	reportMessage := fmt.Sprintf("%s (%s)", messageRemoveWhitespace, ct)
	w.addErrorWithMessageAndFix(start, start, end, reportMessage, []byte{})
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

// hasSelectorCall checks if node contains a selector call with one of the given names.
func hasSelectorCall(node ast.Node, selectorNames []string) bool {
	var found bool

	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false // Already found
		}

		if _, ok := n.(*ast.BlockStmt); ok {
			return false
		}

		if sel, ok := n.(*ast.SelectorExpr); ok {
			found = slices.Contains(selectorNames, sel.Sel.Name)
			return false
		}

		return true
	})

	return found
}

func (w *WSL) isLockOrUnlock(current, previous ast.Node) bool {
	// If we're an ExprStmt (e.g. X()), we check if we're calling `Unlock` or
	// `RWUnlock`. No matter how deep this is or what previous statement was, we
	// allow this.
	//
	// mu.Lock()
	// [ANY BLOCK]
	// mu.Unlock()
	if _, ok := current.(*ast.ExprStmt); ok {
		return hasSelectorCall(current, []string{"Unlock", "RWUnlock"})
	}

	if previous != nil {
		return hasSelectorCall(previous, []string{"Lock", "RWLock", "TryLock"})
	}

	return false
}

// isErrNotNilCheck returns the error identifier if stmt is an `if err != nil`
// or `if err == nil` check without an init statement, nil otherwise.
func (w *WSL) isErrNotNilCheck(stmt ast.Node) *ast.Ident {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok {
		return nil
	}

	// If the error checking has an init condition (e.g. if err := f();) we
	// don't consider it an error check since the error is assigned on this row.
	if ifStmt.Init != nil {
		return nil
	}

	// The condition must be a binary expression (X OP Y)
	binaryExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	// We must do not equal or equal comparison (!= or ==)
	if binaryExpr.Op != token.NEQ && binaryExpr.Op != token.EQL {
		return nil
	}

	xIdent, ok := binaryExpr.X.(*ast.Ident)
	if !ok {
		return nil
	}

	// X is not an error so it's not error checking
	if !w.implementsErr(xIdent) {
		return nil
	}

	yIdent, ok := binaryExpr.Y.(*ast.Ident)
	if !ok {
		return nil
	}

	// Y is not compared with `nil`
	if yIdent.Name != "nil" {
		return nil
	}

	return xIdent
}
