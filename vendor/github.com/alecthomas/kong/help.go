package kong

import (
	"bytes"
	"fmt"
	"go/doc"
	"io"
	"strings"
)

const (
	defaultIndent        = 2
	defaultColumnPadding = 4
)

// Help flag.
type helpValue bool

func (h helpValue) BeforeApply(ctx *Context) error {
	options := ctx.Kong.helpOptions
	options.Summary = false
	err := ctx.Kong.help(options, ctx)
	if err != nil {
		return err
	}
	ctx.Kong.Exit(0)
	return nil
}

// HelpOptions for HelpPrinters.
type HelpOptions struct {
	// Don't print top-level usage summary.
	NoAppSummary bool

	// Write a one-line summary of the context.
	Summary bool

	// Write help in a more compact, but still fully-specified, form.
	Compact bool

	// Tree writes command chains in a tree structure instead of listing them separately.
	Tree bool

	// Indenter modulates the given prefix for the next layer in the tree view.
	// The following exported templates can be used: kong.SpaceIndenter, kong.LineIndenter, kong.TreeIndenter
	// The kong.SpaceIndenter will be used by default.
	Indenter HelpIndenter
}

// Apply options to Kong as a configuration option.
func (h HelpOptions) Apply(k *Kong) error {
	k.helpOptions = h
	return nil
}

// HelpProvider can be implemented by commands/args to provide detailed help.
type HelpProvider interface {
	// This string is formatted by go/doc and thus has the same formatting rules.
	Help() string
}

// HelpIndenter is used to indent new layers in the help tree.
type HelpIndenter func(prefix string) string

// HelpPrinter is used to print context-sensitive help.
type HelpPrinter func(options HelpOptions, ctx *Context) error

// HelpValueFormatter is used to format the help text of flags and positional arguments.
type HelpValueFormatter func(value *Value) string

// DefaultHelpValueFormatter is the default HelpValueFormatter.
func DefaultHelpValueFormatter(value *Value) string {
	if value.Tag.Env == "" {
		return value.Help
	}
	suffix := "($" + value.Tag.Env + ")"
	switch {
	case strings.HasSuffix(value.Help, "."):
		return value.Help[:len(value.Help)-1] + " " + suffix + "."
	case value.Help == "":
		return suffix
	default:
		return value.Help + " " + suffix
	}
}

// DefaultHelpPrinter is the default HelpPrinter.
func DefaultHelpPrinter(options HelpOptions, ctx *Context) error {
	if ctx.Empty() {
		options.Summary = false
	}
	w := newHelpWriter(ctx, options)
	selected := ctx.Selected()
	if selected == nil {
		printApp(w, ctx.Model)
	} else {
		printCommand(w, ctx.Model, selected)
	}
	return w.Write(ctx.Stdout)
}

func printApp(w *helpWriter, app *Application) {
	if !w.NoAppSummary {
		w.Printf("Usage: %s%s", app.Name, app.Summary())
	}
	printNodeDetail(w, app.Node, true)
	cmds := app.Leaves(true)
	if len(cmds) > 0 && app.HelpFlag != nil {
		w.Print("")
		if w.Summary {
			w.Printf(`Run "%s --help" for more information.`, app.Name)
		} else {
			w.Printf(`Run "%s <command> --help" for more information on a command.`, app.Name)
		}
	}
}

func printCommand(w *helpWriter, app *Application, cmd *Command) {
	if !w.NoAppSummary {
		w.Printf("Usage: %s %s", app.Name, cmd.Summary())
	}
	printNodeDetail(w, cmd, true)
	if w.Summary && app.HelpFlag != nil {
		w.Print("")
		w.Printf(`Run "%s --help" for more information.`, cmd.FullPath())
	}
}

func printNodeDetail(w *helpWriter, node *Node, hide bool) {
	if node.Help != "" {
		w.Print("")
		w.Wrap(node.Help)
	}
	if w.Summary {
		return
	}
	if node.Detail != "" {
		w.Print("")
		w.Wrap(node.Detail)
	}
	if len(node.Positional) > 0 {
		w.Print("")
		w.Print("Arguments:")
		writePositionals(w.Indent(), node.Positional)
	}
	if flags := node.AllFlags(true); len(flags) > 0 {
		w.Print("")
		w.Print("Flags:")
		writeFlags(w.Indent(), flags)
	}
	cmds := node.Leaves(hide)
	if len(cmds) > 0 {
		w.Print("")
		w.Print("Commands:")
		if w.Tree {
			writeCommandTree(w, node)
		} else {
			iw := w.Indent()
			if w.Compact {
				writeCompactCommandList(cmds, iw)
			} else {
				writeCommandList(cmds, iw)
			}
		}
	}
}

func writeCommandList(cmds []*Node, iw *helpWriter) {
	for i, cmd := range cmds {
		if cmd.Hidden {
			continue
		}
		printCommandSummary(iw, cmd)
		if i != len(cmds)-1 {
			iw.Print("")
		}
	}
}

func writeCompactCommandList(cmds []*Node, iw *helpWriter) {
	rows := [][2]string{}
	for _, cmd := range cmds {
		if cmd.Hidden {
			continue
		}
		rows = append(rows, [2]string{cmd.Path(), cmd.Help})
	}
	writeTwoColumns(iw, rows)
}

func writeCommandTree(w *helpWriter, node *Node) {
	iw := w.Indent()
	rows := make([][2]string, 0, len(node.Children)*2)
	for i, cmd := range node.Children {
		if cmd.Hidden {
			continue
		}
		rows = append(rows, w.CommandTree(cmd, "")...)
		if i != len(node.Children)-1 {
			rows = append(rows, [2]string{"", ""})
		}
	}
	writeTwoColumns(iw, rows)
}

// nolint: unused
type helpCommandGroup struct {
	Name     string
	Commands []*Node
}

// nolint: unused, deadcode
func collectCommandGroups(nodes []*Node) []helpCommandGroup {
	groups := map[string][]*Node{}
	for _, node := range nodes {
		groups[node.Group] = append(groups[node.Group], node)
	}
	out := []helpCommandGroup{}
	for name, nodes := range groups {
		if name == "" {
			name = "Commands"
		}
		out = append(out, helpCommandGroup{Name: name, Commands: nodes})
	}
	return out
}

func printCommandSummary(w *helpWriter, cmd *Command) {
	w.Print(cmd.Summary())
	if cmd.Help != "" {
		w.Indent().Wrap(cmd.Help)
	}
}

type helpWriter struct {
	indent        string
	width         int
	lines         *[]string
	helpFormatter HelpValueFormatter
	HelpOptions
}

func newHelpWriter(ctx *Context, options HelpOptions) *helpWriter {
	lines := []string{}
	w := &helpWriter{
		indent:        "",
		width:         guessWidth(ctx.Stdout),
		lines:         &lines,
		helpFormatter: ctx.Kong.helpFormatter,
		HelpOptions:   options,
	}
	return w
}

func (h *helpWriter) Printf(format string, args ...interface{}) {
	h.Print(fmt.Sprintf(format, args...))
}

func (h *helpWriter) Print(text string) {
	*h.lines = append(*h.lines, strings.TrimRight(h.indent+text, " "))
}

func (h *helpWriter) Indent() *helpWriter {
	return &helpWriter{indent: h.indent + "  ", lines: h.lines, width: h.width - 2, HelpOptions: h.HelpOptions, helpFormatter: h.helpFormatter}
}

func (h *helpWriter) String() string {
	return strings.Join(*h.lines, "\n")
}

func (h *helpWriter) Write(w io.Writer) error {
	for _, line := range *h.lines {
		_, err := io.WriteString(w, line+"\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *helpWriter) Wrap(text string) {
	w := bytes.NewBuffer(nil)
	doc.ToText(w, strings.TrimSpace(text), "", "    ", h.width)
	for _, line := range strings.Split(strings.TrimSpace(w.String()), "\n") {
		h.Print(line)
	}
}

func writePositionals(w *helpWriter, args []*Positional) {
	rows := [][2]string{}
	for _, arg := range args {
		rows = append(rows, [2]string{arg.Summary(), w.helpFormatter(arg)})
	}
	writeTwoColumns(w, rows)
}

func writeFlags(w *helpWriter, groups [][]*Flag) {
	rows := [][2]string{}
	haveShort := false
	for _, group := range groups {
		for _, flag := range group {
			if flag.Short != 0 {
				haveShort = true
				break
			}
		}
	}
	for i, group := range groups {
		if i > 0 {
			rows = append(rows, [2]string{"", ""})
		}
		for _, flag := range group {
			if !flag.Hidden {
				rows = append(rows, [2]string{formatFlag(haveShort, flag), w.helpFormatter(flag.Value)})
			}
		}
	}
	writeTwoColumns(w, rows)
}

func writeTwoColumns(w *helpWriter, rows [][2]string) {
	maxLeft := 375 * w.width / 1000
	if maxLeft < 30 {
		maxLeft = 30
	}
	// Find size of first column.
	leftSize := 0
	for _, row := range rows {
		if c := len(row[0]); c > leftSize && c < maxLeft {
			leftSize = c
		}
	}

	offsetStr := strings.Repeat(" ", leftSize+defaultColumnPadding)

	for _, row := range rows {
		buf := bytes.NewBuffer(nil)
		doc.ToText(buf, row[1], "", strings.Repeat(" ", defaultIndent), w.width-leftSize-defaultColumnPadding)
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

		line := fmt.Sprintf("%-*s", leftSize, row[0])
		if len(row[0]) < maxLeft {
			line += fmt.Sprintf("%*s%s", defaultColumnPadding, "", lines[0])
			lines = lines[1:]
		}
		w.Print(line)
		for _, line := range lines {
			w.Printf("%s%s", offsetStr, line)
		}
	}
}

// haveShort will be true if there are short flags present at all in the help. Useful for column alignment.
func formatFlag(haveShort bool, flag *Flag) string {
	flagString := ""
	name := flag.Name
	isBool := flag.IsBool()
	if flag.Short != 0 {
		flagString += fmt.Sprintf("-%c, --%s", flag.Short, name)
	} else {
		if haveShort {
			flagString += fmt.Sprintf("    --%s", name)
		} else {
			flagString += fmt.Sprintf("--%s", name)
		}
	}
	if !isBool {
		flagString += fmt.Sprintf("=%s", flag.FormatPlaceHolder())
	}
	return flagString
}

// CommandTree creates a tree with the given node name as root and its children's arguments and sub commands as leaves.
func (h *HelpOptions) CommandTree(node *Node, prefix string) (rows [][2]string) {
	var nodeName string
	switch node.Type {
	default:
		nodeName += prefix + node.Name
	case ArgumentNode:
		nodeName += prefix + "<" + node.Name + ">"
	}
	rows = append(rows, [2]string{nodeName, node.Help})
	if h.Indenter == nil {
		prefix = SpaceIndenter(prefix)
	} else {
		prefix = h.Indenter(prefix)
	}
	for _, arg := range node.Positional {
		rows = append(rows, [2]string{prefix + arg.Summary(), arg.Help})
	}
	for _, subCmd := range node.Children {
		rows = append(rows, h.CommandTree(subCmd, prefix)...)
	}
	return
}

// SpaceIndenter adds a space indent to the given prefix.
func SpaceIndenter(prefix string) string {
	return prefix + strings.Repeat(" ", defaultIndent)
}

// LineIndenter adds line points to every new indent.
func LineIndenter(prefix string) string {
	if prefix == "" {
		return "- "
	}
	return strings.Repeat(" ", defaultIndent) + prefix
}

// TreeIndenter adds line points to every new indent and vertical lines to every layer.
func TreeIndenter(prefix string) string {
	if prefix == "" {
		return "|- "
	}
	return "|" + strings.Repeat(" ", defaultIndent) + prefix
}
