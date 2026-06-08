package jsonnet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/toolutils"
)

type Debugger struct {
	// VM evaluating the input
	vm *VM

	// Interpreter built by the evaluation. Required to look up variables and stack traces
	interpreter *interpreter

	// breakpoints are stored as the result of the .String function of
	// *ast.LocationRange to speed up lookup
	breakpoints map[string]bool

	// The events channel is used to communicate events happening in the VM with the debugger
	events chan DebugEvent
	// The cont channel is used to pass continuation events from the frontend to the VM
	cont chan continuationEvent

	// lastEvaluation stores the result of the last evaluated node
	lastEvaluation value

	// breakOnNode allows the debugger to request continuation until after a
	// certain node has been evaluated (step-out)
	breakOnNode ast.Node

	// singleStep is used to break on every instruction if set to true
	singleStep bool

	// skip skips all hooks when performing sub-evaluation (to lookup vars)
	skip bool

	// current keeps track of the node currently being evaluated
	current ast.Node
}

// ContinuationEvents are sent by the debugger frontend. Specifying `until`
// results in continuation until the evaluated node matches the argument
type continuationEvent struct {
	until *ast.Node
}

type DebugStopReason int

const (
	StopReasonStep DebugStopReason = iota
	StopReasonBreakpoint
	StopReasonException
)

// A DebugEvent is emitted by the hooks to signal certain events happening in the VM. Examples are:
// - Hitting a breakpoint
// - Catching an exception
// - Program termination
type DebugEvent interface {
	anEvent()
}

type DebugEventExit struct {
	Output string
	Error  error
}

func (d *DebugEventExit) anEvent() {}

type DebugEventStop struct {
	Reason         DebugStopReason
	Breakpoint     string
	Current        ast.Node
	LastEvaluation *string
	Error          error

	// efmt is used to format the error (if any). Built by the vm so we need to
	// keep a reference in the event
	efmt ErrorFormatter
}

func (d *DebugEventStop) anEvent() {}
func (d *DebugEventStop) ErrorFmt() string {
	return d.efmt.Format(d.Error)
}

func MakeDebugger() *Debugger {
	d := &Debugger{
		events: make(chan DebugEvent, 2048),
		cont:   make(chan continuationEvent),
	}
	vm := MakeVM()
	vm.EvalHook = EvalHook{
		pre:  d.preHook,
		post: d.postHook,
	}
	d.vm = vm
	d.breakpoints = make(map[string]bool)
	return d
}

func traverse(root ast.Node, f func(node *ast.Node) error) error {
	if err := f(&root); err != nil {
		return fmt.Errorf("pre error: %w", err)
	}

	children := toolutils.Children(root)
	for _, c := range children {
		if err := traverse(c, f); err != nil {
			return err
		}
	}
	return nil
}

func (d *Debugger) Continue() {
	d.cont <- continuationEvent{}
}
func (d *Debugger) ContinueUntilAfter(n ast.Node) {
	d.cont <- continuationEvent{
		until: &n,
	}
}

func (d *Debugger) Step() {
	d.singleStep = true
	d.Continue()
}

func (d *Debugger) Terminate() {
	d.events <- &DebugEventExit{
		Error: fmt.Errorf("terminated"),
	}
}

func (d *Debugger) postHook(i *interpreter, n ast.Node, v value, err error) {
	d.lastEvaluation = v
	if d.skip {
		return
	}
	if err != nil {
		d.events <- &DebugEventStop{
			Current: n,
			Reason:  StopReasonException,
			Error:   err,
			efmt:    d.vm.ErrorFormatter,
		}
		d.waitForContinuation()
	}
	if d.breakOnNode == n {
		d.breakOnNode = nil
		d.singleStep = true
	}
}

func (d *Debugger) waitForContinuation() {
	c := <-d.cont
	if c.until != nil {
		d.breakOnNode = *c.until
	}
}

func (d *Debugger) preHook(i *interpreter, n ast.Node) {
	d.interpreter = i
	d.current = n
	if d.skip {
		return
	}

	switch n.(type) {
	case *ast.LiteralNull, *ast.LiteralNumber, *ast.LiteralString, *ast.LiteralBoolean:
		return
	}
	l := n.Loc()
	if l.File == nil {
		return
	}
	vs := debugValueToString(d.lastEvaluation)
	if d.singleStep {
		d.singleStep = false
		d.events <- &DebugEventStop{
			Reason:         StopReasonStep,
			Current:        n,
			LastEvaluation: &vs,
		}
		d.waitForContinuation()
		return
	}
	loc := n.Loc()
	if loc == nil || loc.File == nil {
		// virtual file such as <std>
		return
	}
	if _, ok := d.breakpoints[loc.String()]; ok {
		d.events <- &DebugEventStop{
			Reason:         StopReasonBreakpoint,
			Breakpoint:     loc.Begin.String(),
			Current:        n,
			LastEvaluation: &vs,
		}
		d.waitForContinuation()
	}
	return
}

func (d *Debugger) ActiveBreakpoints() []string {
	bps := []string{}
	for k := range d.breakpoints {
		bps = append(bps, k)
	}
	return bps
}

func (d *Debugger) BreakpointLocations(file string) ([]*ast.LocationRange, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	a, err := SnippetToAST(file, string(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid source file: %w", err)
	}
	bps := []*ast.LocationRange{}
	traverse(a, func(n *ast.Node) error {
		if n != nil {
			l := (*n).Loc()
			if l.File != nil {
				bps = append(bps, l)
			}
		}
		return nil
	})
	return bps, nil
}

func (d *Debugger) SetBreakpoint(file string, line int, column int) (string, error) {
	valid, err := d.BreakpointLocations(file)
	if err != nil {
		return "", fmt.Errorf("getting valid breakpoint locations: %w", err)
	}
	target := ""
	for _, b := range valid {
		if b.Begin.Line == line {
			if column < 0 {
				target = b.String()
				break
			} else if b.Begin.Column == column {
				target = b.String()
				break
			}
		}
	}
	if target == "" {
		return "", fmt.Errorf("breakpoint location invalid")
	}
	d.breakpoints[target] = true
	return target, nil
}
func (d *Debugger) ClearBreakpoints(file string) {
	abs, _ := filepath.Abs(file)
	for k := range d.breakpoints {
		parts := strings.Split(k, ":")
		full, err := filepath.Abs(parts[0])
		if err == nil && full == abs {
			delete(d.breakpoints, k)
		}
	}
}

func (d *Debugger) LookupValue(val string) (string, error) {
	switch val {
	case "self":
		return debugValueToString(d.interpreter.stack.getSelfBinding().self), nil
	case "super":
		return debugValueToString(d.interpreter.stack.getSelfBinding().super().self), nil
	default:
		v := d.interpreter.stack.lookUpVar(ast.Identifier(val))
		if v != nil {
			if v.content == nil {
				d.skip = true
				e, err := func() (rv value, err error) { // closure to use defer->recover
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("%v", r)
						}
					}()
					rv, err = d.interpreter.rawevaluate(v.body, 0)
					return
				}()
				d.skip = false
				if err != nil {
					return "", err
				}
				v.content = e
			}
			return debugValueToString(v.content), nil
		}
	}
	return "", fmt.Errorf("invalid identifier %s", val)
}

func (d *Debugger) ListVars() []ast.Identifier {
	if d.interpreter != nil {
		return d.interpreter.stack.listVars()
	}
	return make([]ast.Identifier, 0)
}

func (d *Debugger) Launch(filename, snippet string, jpaths []string) {
	jpaths = append(jpaths, filepath.Dir(filename))
	d.vm.Importer(&FileImporter{
		JPaths: jpaths,
	})
	go func() {
		out, err := d.vm.EvaluateAnonymousSnippet(filename, snippet)
		d.events <- &DebugEventExit{
			Output: out,
			Error:  err,
		}
	}()
}

func (d *Debugger) Events() chan DebugEvent {
	return d.events
}

func (d *Debugger) StackTrace() []TraceFrame {
	if d.interpreter == nil || d.current == nil {
		return nil
	}
	trace := d.interpreter.getCurrentStackTrace()
	for i, t := range trace {
		trace[i].Name = t.Loc.FileName // use pseudo file name as name
	}
	trace[len(trace)-1].Loc = *d.current.Loc()
	return trace
}

func debugValueToString(v value) string {
	switch i := v.(type) {
	case *valueFlatString:
		return "\"" + i.getGoString() + "\""
	case *valueObject:
		if i == nil {
			return "{}"
		}
		var sb strings.Builder
		sb.WriteString("{")
		firstLine := true
		for k, v := range i.cache {
			if k.depth != 0 {
				continue
			}
			if !firstLine {
				sb.WriteString(", ")
				firstLine = true
			}
			sb.WriteString(k.field)
			sb.WriteString(": ")
			sb.WriteString(debugValueToString(v))
		}
		sb.WriteString("}")
		return sb.String()
	case *valueArray:
		var sb strings.Builder
		sb.WriteString("[")
		for i, e := range i.elements {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(debugValueToString(e.content))
		}
		sb.WriteString("]")
		return sb.String()
	case *valueNumber:
		return fmt.Sprintf("%f", i.value)
	case *valueBoolean:
		return fmt.Sprintf("%t", i.value)
	case *valueFunction:
		var sb strings.Builder
		sb.WriteString("function(")
		for i, p := range i.parameters() {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(string(p.name))
		}
		sb.WriteString(")")
		return sb.String()
	}
	return fmt.Sprintf("%T%+v", v, v)
}
