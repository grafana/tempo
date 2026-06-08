package types

import "fmt"

type stronglyConnectedComponentID int

// XXX(sbarzowski) put graph transformation here
// XXX(sbarzowski) write "table of contents" explaining what is where
func (g *typeGraph) getOrCreateElementType(target placeholderID, index *indexSpec) (bool, placeholderID) {
	// In case there was no previous indexing
	if g.elementType[target] == nil {
		g.elementType[target] = &elementDesc{}
	}

	elementType := g.elementType[target]

	created := false

	// Actual specific indexing depending on the index type
	if index.indexType == knownStringIndex {
		if elementType.knownStringIndex == nil {
			elementType.knownStringIndex = make(map[string]placeholderID)
		}
		if elementType.knownStringIndex[index.knownStringIndex] == noType {
			created = true
			elID := g.newPlaceholder()
			elementType.knownStringIndex[index.knownStringIndex] = elID
			return created, elID
		}
		return created, elementType.knownStringIndex[index.knownStringIndex]
	} else if index.indexType == knownIntIndex {
		if elementType.knownIntIndex == nil {
			elementType.knownIntIndex = make([]placeholderID, maxKnownCount)
		}
		if elementType.knownIntIndex[index.knownIntIndex] == noType {
			created = true
			elID := g.newPlaceholder()
			elementType.knownIntIndex[index.knownIntIndex] = elID
			return created, elID
		}
		return created, elementType.knownIntIndex[index.knownIntIndex]
	} else if index.indexType == functionIndex {
		if elementType.callIndex == noType {
			created = true
			elementType.callIndex = g.newPlaceholder()
		}
		return created, elementType.callIndex
	} else if index.indexType == genericIndex {
		if elementType.genericIndex == noType {
			created = true
			elementType.genericIndex = g.newPlaceholder()
		}
		return created, elementType.genericIndex
	} else {
		panic("unknown index type")
	}
}

func (g *typeGraph) setElementType(target placeholderID, index *indexSpec, newID placeholderID) {
	elementType := g.elementType[target]

	if index.indexType == knownStringIndex {
		elementType.knownStringIndex[index.knownStringIndex] = newID
	} else if index.indexType == functionIndex {
		elementType.callIndex = newID
	} else {
		elementType.genericIndex = newID
	}
}

// simplifyReferences removes indirection through simple references, i.e. placeholders which contain
// exactly one other placeholder and which don't add anything else.
func (g *typeGraph) simplifyReferences() {
	mapping := make([]placeholderID, len(g._placeholders))
	for i, p := range g._placeholders {
		if p.concrete.Void() && p.index == nil && p.builtinOp == nil && len(p.contains) == 1 {
			mapping[i] = p.contains[0]
		} else {
			mapping[i] = placeholderID(i)
		}
	}

	// transitive closure
	for i := range mapping {
		if mapping[mapping[i]] != mapping[i] {
			mapping[i] = mapping[mapping[i]]
		}
	}

	for i := range g._placeholders {
		p := g.placeholder(placeholderID(i))
		for j := range p.contains {
			p.contains[j] = mapping[p.contains[j]]
		}
		if p.index != nil {
			p.index.indexed = mapping[p.index.indexed]
		}
	}

	for k := range g.exprPlaceholder {
		g.exprPlaceholder[k] = mapping[g.exprPlaceholder[k]]
	}
}

func (g *typeGraph) separateElementTypes() {
	var getElementType func(container placeholderID, index *indexSpec) placeholderID
	getElementType = func(container placeholderID, index *indexSpec) placeholderID {
		c := g.placeholder(container)
		created, elID := g.getOrCreateElementType(container, index)

		if !created {
			return elID
		}

		// Builtins
		// Simple "know nothing" upper bound for the arguments
		var fromBuiltin builtinOpResult
		if c.builtinOp != nil {
			fromBuiltin = c.builtinOp.withUnknown()
		}

		// We can have concrete values either directly associated with the placeholder
		// or coming from the builtin (some builtins may have a known result type even
		// with unknown arguments).
		concrete := c.concrete
		// TODO(sbarzowski) Add tests when relevant. Currently the only builtin we have
		// here is plus which does not provide any concrete result for unknown arguments.
		concrete.widen(&fromBuiltin.concrete)

		// Now we need to put all the stuff into element type
		contains := make([]placeholderID, 0, 1)

		// Direct indexing
		if index.indexType == knownStringIndex {
			if concrete.Object() {
				if ps, present := c.concrete.ObjectDesc.fieldContains[index.knownStringIndex]; present {
					contains = append(contains, ps...)
				} else if !c.concrete.ObjectDesc.allFieldsKnown {
					contains = append(contains, c.concrete.ObjectDesc.unknownContain...)
				}
			}
		} else if index.indexType == knownIntIndex {
			if concrete.Array() {
				if index.knownIntIndex < len(c.concrete.ArrayDesc.elementContains) {
					contains = append(contains, c.concrete.ArrayDesc.elementContains[index.knownIntIndex]...)
				} else {
					contains = append(contains, c.concrete.ArrayDesc.furtherContain...)
				}
			}

			if c.concrete.String {
				contains = append(contains, stringType)
			}
		} else if index.indexType == functionIndex {
			if concrete.Function() {
				contains = append(contains, c.concrete.FunctionDesc.resultContains...)
			}
		} else if index.indexType == genericIndex {
			// TODO(sbarzowski) performance issues when the object is big
			if concrete.Object() {
				contains = append(contains, c.concrete.ObjectDesc.unknownContain...)
				for _, placeholders := range c.concrete.ObjectDesc.fieldContains {
					contains = append(contains, placeholders...)
				}
			}

			if concrete.ArrayDesc != nil {
				for _, placeholders := range c.concrete.ArrayDesc.elementContains {
					contains = append(contains, placeholders...)
				}
				contains = append(contains, c.concrete.ArrayDesc.furtherContain...)
			}

			if concrete.String {
				contains = append(contains, stringType)
			}
		} else {
			panic("unknown index type")
		}

		// The indexed thing may itself be indexing something, so we need to go deeper
		if c.index != nil {
			elInC := getElementType(c.index.indexed, c.index)
			contains = append(contains, getElementType(elInC, index))
		}

		// The indexed thing may contain other values, we need to index those as well
		for _, contained := range c.contains {
			contains = append(contains, getElementType(contained, index))
		}
		for _, contained := range fromBuiltin.contained {
			contains = append(contains, getElementType(contained, index))
		}

		contains = normalizePlaceholders(contains)
		g._placeholders[elID].contains = contains

		// Immediate path compression
		if len(contains) == 1 {
			g.setElementType(container, index, contains[0])
			return contains[0]
		}

		return elID
	}

	for i := range g._placeholders {
		index := g.placeholder(placeholderID(i)).index
		if index != nil {
			el := getElementType(index.indexed, index)
			// We carefully take a new pointer here, because getElementType might have reallocated it
			tp := &g._placeholders[i]
			tp.index = nil
			tp.contains = append(tp.contains, el)
		}
	}
}

func (g *typeGraph) makeTopoOrder() {
	visited := make([]bool, len(g._placeholders))

	g.topoOrder = make([]placeholderID, 0, len(g._placeholders))

	var visit func(p placeholderID)
	visit = func(p placeholderID) {
		visited[p] = true
		node := g.placeholder(p)
		for _, child := range node.contains {
			if !visited[child] {
				visit(child)
			}
		}
		if node.builtinOp != nil {
			for _, child := range node.builtinOp.args {
				if !visited[child] {
					visit(child)
				}
			}
		}
		g.topoOrder = append(g.topoOrder, p)
	}

	for i := range g._placeholders {
		if !visited[i] {
			visit(placeholderID(i))
		}
	}
}

func (g *typeGraph) findTypes() {
	dependentOn := make([][]placeholderID, len(g._placeholders))
	for i, p := range g._placeholders {
		for _, dependency := range p.contains {
			dependentOn[dependency] = append(dependentOn[dependency], placeholderID(i))
		}
		if p.builtinOp != nil {
			for _, dependency := range p.builtinOp.args {
				dependentOn[dependency] = append(dependentOn[dependency], placeholderID(i))
			}
		}
	}

	visited := make([]bool, len(g._placeholders))
	g.sccOf = make([]stronglyConnectedComponentID, len(g._placeholders))

	stronglyConnectedComponents := make([][]placeholderID, 0)
	var sccID stronglyConnectedComponentID

	var visit func(p placeholderID)
	visit = func(p placeholderID) {
		visited[p] = true
		g.sccOf[p] = sccID
		stronglyConnectedComponents[sccID] = append(stronglyConnectedComponents[sccID], p)
		for _, dependent := range dependentOn[p] {
			if !visited[dependent] {
				visit(dependent)
			}
		}
	}

	g.upperBound = make([]TypeDesc, len(g._placeholders))

	for i := len(g.topoOrder) - 1; i >= 0; i-- {
		p := g.topoOrder[i]
		if !visited[p] {
			stronglyConnectedComponents = append(stronglyConnectedComponents, make([]placeholderID, 0, 1))
			visit(p)
			sccID++
		}
	}

	for i := len(stronglyConnectedComponents) - 1; i >= 0; i-- {
		scc := stronglyConnectedComponents[i]
		g.resolveTypesInSCC(scc)
	}
}

func (g *typeGraph) resolveTypesInSCC(scc []placeholderID) {
	sccID := g.sccOf[scc[0]]

	common := voidTypeDesc()

	for _, p := range scc {
		for _, contained := range g.placeholder(p).contains {
			if g.sccOf[contained] != sccID {
				common.widen(&g.upperBound[contained])
			}
		}
		builtinOp := g.placeholder(p).builtinOp
		if builtinOp != nil {
			concreteArgs := []*TypeDesc{}
			for _, arg := range builtinOp.args {
				if g.sccOf[arg] != sccID {
					concreteArgs = append(concreteArgs, &g.upperBound[arg])
				} else {
					concreteArgs = append(concreteArgs, nil)
				}
			}
			res := builtinOp.f(concreteArgs, builtinOp.args)
			common.widen(&res.concrete)
			for _, contained := range res.contained {
				if g.sccOf[contained] != sccID {
					common.widen(&g.upperBound[contained])
				}
			}
		}
	}

	for _, p := range scc {
		common.widen(&g.placeholder(p).concrete)
		if g.placeholder(p).index != nil {
			panic(fmt.Sprintf("All indexing should have been rewritten to direct references at this point (indexing %d, indexed %d)", p, g.placeholder(p).index.indexed))
		}
	}

	common.normalize()

	for _, p := range scc {
		g.upperBound[p] = common
	}
}
