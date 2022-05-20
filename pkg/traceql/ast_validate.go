package traceql

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
	err := o.lhs.validate()
	if err != nil {
		return err
	}
	return o.rhs.validate()
}

func (a Aggregate) validate() error {
	if a.e == nil {
		return nil
	}

	return a.e.validate()
}

func (o SpansetOperation) validate() error {
	err := o.lhs.validate()
	if err != nil {
		return err
	}
	return o.rhs.validate()
}

func (f SpansetFilter) validate() error {
	return f.e.validate()
}

func (f ScalarFilter) validate() error {
	err := f.lhs.validate()
	if err != nil {
		return err
	}
	return f.rhs.validate()
}

func (o BinaryOperation) validate() error {
	err := o.lhs.validate()
	if err != nil {
		return err
	}
	return o.rhs.validate()
}

func (o UnaryOperation) validate() error {
	return o.e.validate()
}

func (n Static) validate() error {
	return nil
}

func (a Attribute) validate() error {
	return nil
}
