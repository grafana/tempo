package traceql

import "fmt"

func (r RootExpr) validate() error {
	return r.p.validate()
}

func (p Pipeline) validate() error {
	for _, p := range p.p {
		err := p.validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o GroupOperation) validate() error {
	return o.e.validate()
}

func (o CoalesceOperation) validate() error {
	return nil
}

func (o ScalarOperation) validate() error {
	if err := o.lhs.validate(); err != nil {
		return err
	}
	if err := o.rhs.validate(); err != nil {
		return err
	}

	lhsT := o.lhs.impliedType()
	rhsT := o.rhs.impliedType()
	if !lhsT.isMatchingOperand(rhsT) {
		return fmt.Errorf("binary operations must operate on the same type: %s", o.String())
	}

	if !o.op.binaryTypesValid(lhsT, rhsT) {
		return fmt.Errorf("illegal operation for the given types: %s", o.String())
	}

	return nil
}

func (a Aggregate) validate() error {
	if a.e == nil {
		return nil
	}

	if err := a.e.validate(); err != nil {
		return err
	}

	// aggregate field expressions require a type of a number or attribute
	t := a.e.impliedType()
	if t != typeAttribute && !t.isNumeric() {
		return fmt.Errorf("aggregate field expressions must resolve to a number type: %s", a.String())
	}

	return nil
}

func (o SpansetOperation) validate() error {
	if err := o.lhs.validate(); err != nil {
		return err
	}
	return o.rhs.validate()
}

func (f SpansetFilter) validate() error {
	if err := f.e.validate(); err != nil {
		return err
	}

	t := f.e.impliedType()
	if t != typeAttribute && t != typeBoolean {
		return fmt.Errorf("span filter field expressions must resolve to a boolean: %s", f.String())
	}

	return nil
}

func (f ScalarFilter) validate() error {
	if err := f.lhs.validate(); err != nil {
		return err
	}
	if err := f.rhs.validate(); err != nil {
		return err
	}

	lhsT := f.lhs.impliedType()
	rhsT := f.rhs.impliedType()
	if !lhsT.isMatchingOperand(rhsT) {
		return fmt.Errorf("binary operations must operate on the same type: %s", f.String())
	}

	if !f.op.binaryTypesValid(lhsT, rhsT) {
		return fmt.Errorf("illegal operation for the given types: %s", f.String())
	}

	return nil
}

func (o BinaryOperation) validate() error {
	if err := o.lhs.validate(); err != nil {
		return err
	}
	if err := o.rhs.validate(); err != nil {
		return err
	}

	lhsT := o.lhs.impliedType()
	rhsT := o.rhs.impliedType()
	if !lhsT.isMatchingOperand(rhsT) {
		return fmt.Errorf("binary operations must operate on the same type: %s", o.String())
	}

	if !o.op.binaryTypesValid(lhsT, rhsT) {
		return fmt.Errorf("illegal operation for the given types: %s", o.String())
	}

	return nil
}

func (o UnaryOperation) validate() error {
	if err := o.e.validate(); err != nil {
		return err
	}

	t := o.e.impliedType()
	if t == typeAttribute {
		return nil
	}

	if !o.op.unaryTypesValid(t) {
		return fmt.Errorf("illegal operation for the given type: %s", o.String())
	}

	return nil
}

func (n Static) validate() error {
	return nil
}

func (a Attribute) validate() error {
	return nil
}
