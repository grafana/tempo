package traceql

func (f SpansetFilter) extractConditions(request *FetchSpansRequest) {
	f.Expression.extractConditions(request)
}

// extractConditions on Select puts its conditions into the SecondPassConditions
func (o SelectOperation) extractConditions(request *FetchSpansRequest) {
	selectR := &FetchSpansRequest{}
	for _, expr := range o.exprs {
		expr.extractConditions(selectR)
	}
	// copy any conditions to the normal request's SecondPassConditions
	request.SecondPassConditions = append(request.SecondPassConditions, selectR.Conditions...)
}

func (o BinaryOperation) extractConditions(request *FetchSpansRequest) {
	// TODO we can further optimise this by attempting to execute every FieldExpression, if they only contain statics it should resolve
	switch o.LHS.(type) {
	case Attribute:
		switch o.RHS.(type) {
		case Static:
			request.appendCondition(Condition{
				Attribute: o.LHS.(Attribute),
				Op:        o.Op,
				Operands:  []Static{o.RHS.(Static)},
			})
		case Attribute:
			// Both sides are attributes, just fetch both
			request.appendCondition(Condition{
				Attribute: o.LHS.(Attribute),
				Op:        OpNone,
				Operands:  nil,
			})
			request.appendCondition(Condition{
				Attribute: o.RHS.(Attribute),
				Op:        OpNone,
				Operands:  nil,
			})
		default:
			// Just fetch LHS and try to do something smarter with RHS
			request.appendCondition(Condition{
				Attribute: o.LHS.(Attribute),
				Op:        OpNone,
				Operands:  nil,
			})
			o.RHS.extractConditions(request)
		}
	case Static:
		switch o.RHS.(type) {
		case Static:
			// 2 statics, don't need to send any conditions
			return
		case Attribute:
			request.appendCondition(Condition{
				Attribute: o.RHS.(Attribute),
				Op:        o.Op,
				Operands:  []Static{o.LHS.(Static)},
			})
		default:
			o.RHS.extractConditions(request)
		}
	default:
		o.LHS.extractConditions(request)
		o.RHS.extractConditions(request)
		request.AllConditions = request.AllConditions && (o.Op != OpOr)
	}
}

func (o UnaryOperation) extractConditions(request *FetchSpansRequest) {
	// TODO when Op is Not we should just either negate all inner Operands or just fetch the columns with OpNone
	o.Expression.extractConditions(request)
}

func (s Static) extractConditions(*FetchSpansRequest) {
}

func (a Attribute) extractConditions(request *FetchSpansRequest) {
	request.appendCondition(Condition{
		Attribute: a,
		Op:        OpNone,
		Operands:  nil,
	})
}
