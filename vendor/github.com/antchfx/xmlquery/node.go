package xmlquery

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
)

// A NodeType is the type of a Node.
type NodeType uint

const (
	// DocumentNode is a document object that, as the root of the document tree,
	// provides access to the entire XML document.
	DocumentNode NodeType = iota
	// DeclarationNode is the document type declaration, indicated by the
	// following tag (for example, <!DOCTYPE...> ).
	DeclarationNode
	// ElementNode is an element (for example, <item> ).
	ElementNode
	// TextNode is the text content of a node.
	TextNode
	// CharDataNode node <![CDATA[content]]>
	CharDataNode
	// CommentNode a comment (for example, <!-- my comment --> ).
	CommentNode
	// AttributeNode is an attribute of element.
	AttributeNode
	// NotationNode is a directive represents in document (for example, <!text...>).
	NotationNode
)

type Attr struct {
	Name         xml.Name
	Value        string
	NamespaceURI string
}

// A Node consists of a NodeType and some Data (tag name for
// element nodes, content for text) and are part of a tree of Nodes.
type Node struct {
	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Node

	Type         NodeType
	Data         string
	Prefix       string
	NamespaceURI string
	Attr         []Attr

	level int // node level in the tree
}

type outputConfiguration struct {
	printSelf              bool
	preserveSpaces         bool
	emptyElementTagSupport bool
	skipComments           bool
	useIndentation         string
}

type OutputOption func(*outputConfiguration)

// WithOutputSelf configures the Node to print the root node itself
func WithOutputSelf() OutputOption {
	return func(oc *outputConfiguration) {
		oc.printSelf = true
	}
}

// WithEmptyTagSupport empty tags should be written as <empty/> and
// not as <empty></empty>
func WithEmptyTagSupport() OutputOption {
	return func(oc *outputConfiguration) {
		oc.emptyElementTagSupport = true
	}
}

// WithoutComments will skip comments in output
func WithoutComments() OutputOption {
	return func(oc *outputConfiguration) {
		oc.skipComments = true
	}
}

// WithPreserveSpace will preserve spaces in output
func WithPreserveSpace() OutputOption {
	return func(oc *outputConfiguration) {
		oc.preserveSpaces = true
	}
}

// WithIndentation sets the indentation string used for formatting the output.
func WithIndentation(indentation string) OutputOption {
	return func(oc *outputConfiguration) {
		oc.useIndentation = indentation
	}
}

func newXMLName(name string) xml.Name {
	if i := strings.IndexByte(name, ':'); i > 0 {
		return xml.Name{
			Space: name[:i],
			Local: name[i+1:],
		}
	}
	return xml.Name{
		Local: name,
	}
}

func (n *Node) Level() int {
	return n.level
}

// InnerText returns the text between the start and end tags of the object.
func (n *Node) InnerText() string {
	var output func(*strings.Builder, *Node)
	output = func(b *strings.Builder, n *Node) {
		switch n.Type {
		case TextNode, CharDataNode:
			b.WriteString(n.Data)
		case CommentNode:
		default:
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				output(b, child)
			}
		}
	}

	var b strings.Builder
	output(&b, n)
	return b.String()
}

func (n *Node) sanitizedData(preserveSpaces bool) string {
	if preserveSpaces {
		return n.Data
	}
	return strings.TrimSpace(n.Data)
}

func calculatePreserveSpaces(n *Node, pastValue bool) bool {
	if attr := n.SelectAttr("xml:space"); attr == "preserve" {
		return true
	} else if attr == "default" {
		return false
	}
	return pastValue
}

type indentation struct {
	level    int
	hasChild bool
	indent   string
	w io.Writer
}

func newIndentation(indent string, w io.Writer) *indentation {
	if indent == "" {
		return nil
	}
	return &indentation{
		indent: indent,
		w:      w,
	}
}

func (i *indentation) NewLine() {
	if i == nil {
		return
	}
	io.WriteString(i.w, "\n")
}

func (i *indentation) Open() {
	if i == nil {
		return
	}

	io.WriteString(i.w, "\n")
	io.WriteString(i.w, strings.Repeat(i.indent, i.level))

	i.level++
	i.hasChild = false
}

func (i *indentation) Close() {
	if i == nil {
		return
	}
	i.level--
	if i.hasChild {
		io.WriteString(i.w, "\n")
		io.WriteString(i.w, strings.Repeat(i.indent, i.level))
	}
	i.hasChild = true
}

func outputXML(w io.Writer, n *Node, preserveSpaces bool, config *outputConfiguration, indent *indentation) {
	preserveSpaces = calculatePreserveSpaces(n, preserveSpaces)
	switch n.Type {
	case TextNode:
		io.WriteString(w, html.EscapeString(n.sanitizedData(preserveSpaces)))
		return
	case CharDataNode:
		io.WriteString(w, "<![CDATA[")
		io.WriteString(w, n.Data)
		io.WriteString(w, "]]>")
		return
	case CommentNode:
		if !config.skipComments {
			io.WriteString(w, "<!--")
			io.WriteString(w, n.Data)
			io.WriteString(w, "-->")
		}
		return
	case NotationNode:
		indent.NewLine()
		fmt.Fprintf(w, "<!%s>", n.Data)
		return
	case DeclarationNode:
		io.WriteString(w, "<?" + n.Data)
	default:
		indent.Open()
		if n.Prefix == "" {
			io.WriteString(w, "<" + n.Data)
		} else {
			fmt.Fprintf(w, "<%s:%s", n.Prefix, n.Data)
		}
	}

	for _, attr := range n.Attr {
		if attr.Name.Space != "" {
			fmt.Fprintf(w, ` %s:%s=`, attr.Name.Space, attr.Name.Local)
		} else {
			fmt.Fprintf(w, ` %s=`, attr.Name.Local)
		}

		fmt.Fprintf(w, `"%v"`, html.EscapeString(attr.Value))
	}
	if n.Type == DeclarationNode {
		io.WriteString(w, "?>")
	} else {
		if n.FirstChild != nil || !config.emptyElementTagSupport {
			io.WriteString(w, ">")
		} else {
			io.WriteString(w, "/>")
			indent.Close()
			return
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		outputXML(w, child, preserveSpaces, config, indent)
	}
	if n.Type != DeclarationNode {
		indent.Close()
		if n.Prefix == "" {
			fmt.Fprintf(w, "</%s>", n.Data)
		} else {
			fmt.Fprintf(w, "</%s:%s>", n.Prefix, n.Data)
		}
	}
}

// OutputXML returns the text that including tags name.
func (n *Node) OutputXML(self bool) string {
	if self {
		return n.OutputXMLWithOptions(WithOutputSelf())
	}
	return n.OutputXMLWithOptions()
}

// OutputXMLWithOptions returns the text that including tags name.
func (n *Node) OutputXMLWithOptions(opts ...OutputOption) string {
	var b strings.Builder
	n.WriteWithOptions(&b, opts...)
	return b.String()
}

// Write writes xml to given writer.
func (n *Node) Write(writer io.Writer, self bool) {
	if self {
		n.WriteWithOptions(writer, WithOutputSelf())
	}
	n.WriteWithOptions(writer)
}

// WriteWithOptions writes xml with given options to given writer.
func (n *Node) WriteWithOptions(writer io.Writer, opts ...OutputOption) {
	config := &outputConfiguration{}
	// Set the options
	for _, opt := range opts {
		opt(config)
	}
	pastPreserveSpaces := config.preserveSpaces
	preserveSpaces := calculatePreserveSpaces(n, pastPreserveSpaces)
	b := bufio.NewWriter(writer)
	defer b.Flush()

	if config.printSelf && n.Type != DocumentNode {
		outputXML(b, n, preserveSpaces, config, newIndentation(config.useIndentation, b))
	} else {
		for n := n.FirstChild; n != nil; n = n.NextSibling {
			outputXML(b, n, preserveSpaces, config, newIndentation(config.useIndentation, b))
		}
	}
}

// AddAttr adds a new attribute specified by 'key' and 'val' to a node 'n'.
func AddAttr(n *Node, key, val string) {
	attr := Attr{
		Name:  newXMLName(key),
		Value: val,
	}
	n.Attr = append(n.Attr, attr)
}

// SetAttr allows an attribute value with the specified name to be changed.
// If the attribute did not previously exist, it will be created.
func (n *Node) SetAttr(key, value string) {
	name := newXMLName(key)
	for i, attr := range n.Attr {
		if attr.Name == name {
			n.Attr[i].Value = value
			return
		}
	}
	AddAttr(n, key, value)
}

// RemoveAttr removes the attribute with the specified name.
func (n *Node) RemoveAttr(key string) {
	name := newXMLName(key)
	for i, attr := range n.Attr {
		if attr.Name == name {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			return
		}
	}
}

// AddChild adds a new node 'n' to a node 'parent' as its last child.
func AddChild(parent, n *Node) {
	n.Parent = parent
	n.NextSibling = nil
	if parent.FirstChild == nil {
		parent.FirstChild = n
		n.PrevSibling = nil
	} else {
		parent.LastChild.NextSibling = n
		n.PrevSibling = parent.LastChild
	}

	parent.LastChild = n
}

// AddSibling adds a new node 'n' as a sibling of a given node 'sibling'.
// Note it is not necessarily true that the new node 'n' would be added
// immediately after 'sibling'. If 'sibling' isn't the last child of its
// parent, then the new node 'n' will be added at the end of the sibling
// chain of their parent.
func AddSibling(sibling, n *Node) {
	for t := sibling.NextSibling; t != nil; t = t.NextSibling {
		sibling = t
	}
	n.Parent = sibling.Parent
	sibling.NextSibling = n
	n.PrevSibling = sibling
	n.NextSibling = nil
	if sibling.Parent != nil {
		sibling.Parent.LastChild = n
	}
}

// RemoveFromTree removes a node and its subtree from the document
// tree it is in. If the node is the root of the tree, then it's no-op.
func RemoveFromTree(n *Node) {
	if n.Parent == nil {
		return
	}
	if n.Parent.FirstChild == n {
		if n.Parent.LastChild == n {
			n.Parent.FirstChild = nil
			n.Parent.LastChild = nil
		} else {
			n.Parent.FirstChild = n.NextSibling
			n.NextSibling.PrevSibling = nil
		}
	} else {
		if n.Parent.LastChild == n {
			n.Parent.LastChild = n.PrevSibling
			n.PrevSibling.NextSibling = nil
		} else {
			n.PrevSibling.NextSibling = n.NextSibling
			n.NextSibling.PrevSibling = n.PrevSibling
		}
	}
	n.Parent = nil
	n.PrevSibling = nil
	n.NextSibling = nil
}
