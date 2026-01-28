package goconst

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"regexp"
	"strconv"
	"strings"
)

// treeVisitor is used to walk the AST and find strings that could be constants.
type treeVisitor struct {
	fileSet     *token.FileSet
	typeInfo    *types.Info
	packageName string
	p           *Parser
	ignoreRegex *regexp.Regexp
}

// Visit browses the AST tree for strings that could be potentially
// replaced by constants.
// A map of existing constants is built as well (-match-constant).
func (v *treeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}

	// A single case with "ast.BasicLit" would be much easier
	// but then we wouldn't be able to tell in which context
	// the string is defined (could be a constant definition).
	switch t := node.(type) {
	// Scan for constants in an attempt to match strings with existing constants
	case *ast.GenDecl:
		if !v.p.matchConstant && !v.p.findDuplicates {
			return v
		}
		if t.Tok != token.CONST {
			return v
		}

		for _, spec := range t.Specs {
			val := spec.(*ast.ValueSpec)
			for i, str := range val.Values {
				if v.typeInfo != nil && v.p.evalConstExpressions {
					typedVal, ok := v.typeInfo.Types[str]
					if !ok || !v.isSupportedKind(typedVal.Value.Kind()) {
						continue
					}

					v.addConst(val.Names[i].Name, typedVal.Value.String(), str.Pos())
				} else {
					lit, ok := str.(*ast.BasicLit)
					if !ok || !v.isSupported(lit.Kind) {
						continue
					}
					v.addConst(val.Names[i].Name, lit.Value, val.Names[i].Pos())
				}
			}
		}

	// foo := "moo"
	case *ast.AssignStmt:
		for _, rhs := range t.Rhs {
			lit, ok := rhs.(*ast.BasicLit)
			if !ok || !v.isSupported(lit.Kind) {
				continue
			}

			v.addString(lit.Value, rhs.(*ast.BasicLit).Pos(), Assignment)
		}

	// if foo == "moo"
	case *ast.BinaryExpr:
		if t.Op != token.EQL && t.Op != token.NEQ {
			return v
		}

		var lit *ast.BasicLit
		var ok bool

		lit, ok = t.X.(*ast.BasicLit)
		if ok && v.isSupported(lit.Kind) {
			v.addString(lit.Value, lit.Pos(), Binary)
		}

		lit, ok = t.Y.(*ast.BasicLit)
		if ok && v.isSupported(lit.Kind) {
			v.addString(lit.Value, lit.Pos(), Binary)
		}

	// case "foo":
	case *ast.CaseClause:
		for _, item := range t.List {
			lit, ok := item.(*ast.BasicLit)
			if ok && v.isSupported(lit.Kind) {
				v.addString(lit.Value, lit.Pos(), Case)
			}
		}

	// return "boo"
	case *ast.ReturnStmt:
		for _, item := range t.Results {
			lit, ok := item.(*ast.BasicLit)
			if ok && v.isSupported(lit.Kind) {
				v.addString(lit.Value, lit.Pos(), Return)
			}
		}

	// fn("http://")
	case *ast.CallExpr:
		for _, item := range t.Args {
			lit, ok := item.(*ast.BasicLit)
			if ok && v.isSupported(lit.Kind) {
				v.addString(lit.Value, lit.Pos(), Call)
			}
		}
	}

	return v
}

// addString adds a string in the map along with its position in the tree.
func (v *treeVisitor) addString(str string, pos token.Pos, typ Type) {
	// Early type exclusion check
	ok, excluded := v.p.excludeTypes[typ]
	if ok && excluded {
		return
	}

	// Drop quotes if any
	var unquotedStr string
	if strings.HasPrefix(str, `"`) || strings.HasPrefix(str, "`") {
		var err error
		// Reuse strings from pool if possible to avoid allocations
		sb := GetStringBuilder()
		defer PutStringBuilder(sb)

		unquotedStr, err = strconv.Unquote(str)
		if err != nil {
			// If unquoting fails, manually strip quotes
			// This avoids additional temporary strings
			if len(str) >= 2 {
				sb.WriteString(str[1 : len(str)-1])
				unquotedStr = sb.String()
			} else {
				unquotedStr = str
			}
		}
	} else {
		unquotedStr = str
	}

	// Early length check
	if len(unquotedStr) == 0 || len(unquotedStr) < v.p.minLength {
		return
	}

	// Early regex filtering - pre-compiled for efficiency
	if v.ignoreRegex != nil && v.ignoreRegex.MatchString(unquotedStr) {
		return
	}

	// Early number range filtering
	if v.p.numberMin != 0 || v.p.numberMax != 0 {
		if i, err := strconv.ParseInt(unquotedStr, 0, 0); err == nil {
			if (v.p.numberMin != 0 && i < int64(v.p.numberMin)) ||
				(v.p.numberMax != 0 && i > int64(v.p.numberMax)) {
				return
			}
		}
	}

	// Use interned string to reduce memory usage - identical strings share the same memory
	internedStr := InternString(unquotedStr)

	// Update the count first, this is faster than appending to slices
	count := v.p.IncrementStringCount(internedStr)

	// Only continue if we're still adding the position to the map
	// or if count has reached threshold
	if count == 1 || count == v.p.minOccurrences {
		// Lock to safely update the shared map
		v.p.stringMutex.Lock()
		defer v.p.stringMutex.Unlock()

		_, exists := v.p.strs[internedStr]
		if !exists {
			v.p.strs[internedStr] = make([]ExtendedPos, 0, v.p.minOccurrences) // Preallocate with expected size
		}

		// Create an optimized position record
		newPos := ExtendedPos{
			packageName: InternString(v.packageName), // Intern the package name to reduce memory
			Position:    v.fileSet.Position(pos),
		}

		v.p.strs[internedStr] = append(v.p.strs[internedStr], newPos)
	}
}

// addConst adds a const in the map along with its position in the tree.
func (v *treeVisitor) addConst(name string, val string, pos token.Pos) {
	// Early filtering using the same criteria as for strings
	var unquotedVal string
	if strings.HasPrefix(val, `"`) || strings.HasPrefix(val, "`") {
		var err error
		// Use string builder from pool to reduce allocations
		sb := GetStringBuilder()
		defer PutStringBuilder(sb)

		if unquotedVal, err = strconv.Unquote(val); err != nil {
			// If unquoting fails, manually strip quotes without allocations
			if len(val) >= 2 {
				sb.WriteString(val[1 : len(val)-1])
				unquotedVal = sb.String()
			} else {
				unquotedVal = val
			}
		}
	} else {
		unquotedVal = val
	}

	// Skip constants with values that would be filtered anyway
	if len(unquotedVal) < v.p.minLength {
		return
	}

	if v.ignoreRegex != nil && v.ignoreRegex.MatchString(unquotedVal) {
		return
	}

	// Use interned string to reduce memory usage
	internedVal := InternString(unquotedVal)
	internedName := InternString(name)
	internedPkg := InternString(v.packageName)

	// Lock to safely update the shared map
	v.p.constMutex.Lock()
	defer v.p.constMutex.Unlock()

	// track this const if this is a new const, or if we are searching for duplicate consts
	if _, ok := v.p.consts[internedVal]; !ok || v.p.findDuplicates {
		v.p.consts[internedVal] = append(v.p.consts[internedVal], ConstType{
			Name:        internedName,
			packageName: internedPkg,
			Position:    v.fileSet.Position(pos),
		})
	}
}

func (v *treeVisitor) isSupported(tk token.Token) bool {
	for _, s := range v.p.supportedTokens {
		if tk == s {
			return true
		}
	}
	return false
}

func (v *treeVisitor) isSupportedKind(kind constant.Kind) bool {
	for _, s := range v.p.supportedKinds {
		if kind == s {
			return true
		}
	}
	return false
}
