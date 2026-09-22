package backend

import (
	"bytes"
	"fmt"
	"slices"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/grafana/tempo/v3/pkg/tempopb"
)

// DedicatedColumnLayout owns an immutable column configuration and its encoded form.
type (
	DedicatedColumnLayout      struct{ state *dedicatedColumnLayoutState }
	dedicatedColumnLayoutState struct {
		columns DedicatedColumns
		once    sync.Once
		encoded []byte
		err     error
	}
)

func cloneColumns(cols DedicatedColumns) DedicatedColumns {
	cols = slices.Clone(cols)
	for i := range cols {
		cols[i].Options = slices.Clone(cols[i].Options)
	}
	return cols
}

// NewDedicatedColumnLayout copies columns and their options so callers can safely reuse the input.
func NewDedicatedColumnLayout(cols DedicatedColumns) DedicatedColumnLayout {
	if len(cols) == 0 {
		return DedicatedColumnLayout{}
	}
	return DedicatedColumnLayout{state: &dedicatedColumnLayoutState{columns: cloneColumns(cols)}}
}

// Columns returns a copy that can be modified without invalidating the layout.
func (c DedicatedColumnLayout) Columns() DedicatedColumns {
	if c.state == nil {
		return nil
	}
	return cloneColumns(c.state.columns)
}
func (c DedicatedColumnLayout) IsZero() bool { return c.state == nil }
func (c DedicatedColumnLayout) encoded() ([]byte, error) {
	if c.state == nil {
		return nil, nil
	}
	c.state.once.Do(func() { c.state.encoded, c.state.err = sonic.Marshal(c.state.columns) })
	return c.state.encoded, c.state.err
}
func (c DedicatedColumnLayout) Size() int { b, _ := c.encoded(); return len(b) }
func (c DedicatedColumnLayout) MarshalTo(dst []byte) (int, error) {
	b, err := c.encoded()
	if err != nil {
		return 0, err
	}
	return copy(dst, b), nil
}

func (c DedicatedColumnLayout) MarshalJSON() ([]byte, error) {
	if c.state == nil {
		return []byte("null"), nil
	}
	b, err := c.encoded()
	return bytes.Clone(b), err
}

func (c *DedicatedColumnLayout) UnmarshalJSON(data []byte) error {
	if c.state != nil {
		cols := c.Columns()
		type plainColumns DedicatedColumns
		if err := sonic.Unmarshal(data, (*plainColumns)(&cols)); err != nil {
			return err
		}
		*c = DedicatedColumnLayout{}
		if len(cols) > 0 {
			c.state = &dedicatedColumnLayoutState{columns: cols}
		}
		return nil
	}
	if v, ok := getDedicatedColumnLayoutFromCache(data); ok {
		*c = v
		return nil
	}
	type plainColumns DedicatedColumns
	var cols plainColumns
	if err := sonic.Unmarshal(data, &cols); err != nil {
		return err
	}
	*c = DedicatedColumnLayout{}
	if len(cols) > 0 {
		c.state = &dedicatedColumnLayoutState{columns: DedicatedColumns(cols)}
	}
	dedicatedColumnsCache.Set(string(data), *c)
	return nil
}

func (c *DedicatedColumnLayout) Unmarshal(data []byte) error {
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	return c.UnmarshalJSON(data)
}

func (c DedicatedColumnLayout) Hash() uint64 {
	if c.state == nil {
		return 0
	}
	return c.state.columns.Hash()
}

func (c DedicatedColumnLayout) Equal(other DedicatedColumnLayout) bool {
	if c.state == nil || other.state == nil {
		return c.state == other.state
	}
	return slices.EqualFunc(c.state.columns, other.state.columns, func(a, b DedicatedColumn) bool {
		return a.Scope == b.Scope && a.Type == b.Type && a.Name == b.Name && slices.Equal(a.Options, b.Options)
	})
}

// DedicatedColumnsView exposes column values without allowing changes to their owner.
type DedicatedColumnsView interface {
	Len() int
	At(int) DedicatedColumnView
	Hash() uint64
	ToTempopb() ([]*tempopb.DedicatedColumn, error)
}

func (c DedicatedColumnLayout) Len() int {
	if c.state == nil {
		return 0
	}
	return len(c.state.columns)
}

func (c DedicatedColumnLayout) At(i int) DedicatedColumnView {
	return columnView(c.state.columns[i])
}
func (dcs DedicatedColumns) Len() int                     { return len(dcs) }
func (dcs DedicatedColumns) At(i int) DedicatedColumnView { return columnView(dcs[i]) }

func (c DedicatedColumnLayout) ToTempopb() ([]*tempopb.DedicatedColumn, error) {
	if c.state == nil {
		return nil, nil
	}
	return c.state.columns.ToTempopb()
}

// DedicatedColumnView exposes a column without sharing mutable option storage.
type DedicatedColumnView struct {
	Scope   DedicatedColumnScope
	Name    string
	Type    DedicatedColumnType
	options DedicatedColumnOptions
}

func columnView(c DedicatedColumn) DedicatedColumnView {
	return DedicatedColumnView{Scope: c.Scope, Name: c.Name, Type: c.Type, options: c.Options}
}
func (c DedicatedColumnView) HasOptions() bool { return len(c.options) > 0 }
func (c DedicatedColumnView) HasOption(option DedicatedColumnOption) bool {
	return slices.Contains(c.options, option)
}

// Column returns a mutable copy for constructing a different layout.
func (c DedicatedColumnView) Column() DedicatedColumn {
	return DedicatedColumn{Scope: c.Scope, Name: c.Name, Type: c.Type, Options: slices.Clone(c.options)}
}

func (c DedicatedColumnLayout) String() string {
	if c.state == nil {
		return "[]"
	}
	return fmt.Sprintf("%+v", c.state.columns)
}
