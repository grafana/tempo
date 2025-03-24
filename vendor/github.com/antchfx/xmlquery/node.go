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

// WithoutPreserveSpace will not preserve spaces in output
func WithoutPreserveSpace() OutputOption {
	return func(oc *outputConfiguration) {
		oc.preserveSpaces = false
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
	w        io.Writer
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

func (i *indentation) NewLine() (err error) {
	if i == nil {
		return
	}
	_, err = io.WriteString(i.w, "\n")
	return
}

func (i *indentation) Open() (err error) {
	if i == nil {
		return
	}

	if err = i.writeIndent(); err != nil {
		return
	}

	i.level++
	i.hasChild = false
	return
}

func (i *indentation) Close() (err error) {
	if i == nil {
		return
	}
	i.level--
	if i.hasChild {
		if err = i.writeIndent(); err != nil {
			return
		}
	}
	i.hasChild = true
	return
}

func (i *indentation) writeIndent() (err error) {
	_, err = io.WriteString(i.w, "\n")
	if err != nil {
		return
	}
	_, err = io.WriteString(i.w, strings.Repeat(i.indent, i.level))
	return
}

func outputXML(w io.Writer, n *Node, preserveSpaces bool, config *outputConfiguration, indent *indentation) (err error) {
	preserveSpaces = calculatePreserveSpaces(n, preserveSpaces)
	switch n.Type {
	case TextNode:
		_, err = io.WriteString(w, html.EscapeString(n.sanitizedData(preserveSpaces)))
		return
	case CharDataNode:
		_, err = fmt.Fprintf(w, "<![CDATA[%v]]>", n.Data)
		return
	case CommentNode:
		if !config.skipComments {
			_, err = fmt.Fprintf(w, "<!--%v-->", n.Data)
		}
		return
	case NotationNode:
		if err = indent.NewLine(); err != nil {
			return
		}
		_, err = fmt.Fprintf(w, "<!%s>", n.Data)
		return
	case DeclarationNode:
		_, err = io.WriteString(w, "<?"+n.Data)
		if err != nil {
			return
		}
	default:
		if err = indent.Open(); err != nil {
			return
		}
		if n.Prefix == "" {
			_, err = io.WriteString(w, "<"+n.Data)
		} else {
			_, err = fmt.Fprintf(w, "<%s:%s", n.Prefix, n.Data)
		}
		if err != nil {
			return
		}
	}

	for _, attr := range n.Attr {
		if attr.Name.Space != "" {
			_, err = fmt.Fprintf(w, ` %s:%s=`, attr.Name.Space, attr.Name.Local)
		} else {
			_, err = fmt.Fprintf(w, ` %s=`, attr.Name.Local)
		}
		if err != nil {
			return
		}

		_, err = fmt.Fprintf(w, `"%v"`, html.EscapeString(attr.Value))
		if err != nil {
			return
		}
	}
	if n.Type == DeclarationNode {
		_, err = io.WriteString(w, "?>")
	} else {
		if n.FirstChild != nil || !config.emptyElementTagSupport {
			_, err = io.WriteString(w, ">")
		} else {
			_, err = io.WriteString(w, "/>")
			if err != nil {
				return
			}
			err = indent.Close()
			return
		}
	}
	if err != nil {
		return
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		err = outputXML(w, child, preserveSpaces, config, indent)
		if err != nil {
			return
		}
	}
	if n.Type != DeclarationNode {
		if err = indent.Close(); err != nil {
			return
		}
		if n.Prefix == "" {
			_, err = fmt.Fprintf(w, "</%s>", n.Data)
		} else {
			_, err = fmt.Fprintf(w, "</%s:%s>", n.Prefix, n.Data)
		}
	}
	return
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
func (n *Node) Write(writer io.Writer, self bool) error {
	if self {
		return n.WriteWithOptions(writer, WithOutputSelf())
	}
	return n.WriteWithOptions(writer)
}

// WriteWithOptions writes xml with given options to given writer.
func (n *Node) WriteWithOptions(writer io.Writer, opts ...OutputOption) (err error) {
	config := &outputConfiguration{
		preserveSpaces: true,
	}
	// Set the options
	for _, opt := range opts {
		opt(config)
	}
	pastPreserveSpaces := config.preserveSpaces
	preserveSpaces := calculatePreserveSpaces(n, pastPreserveSpaces)
	b := bufio.NewWriter(writer)
	defer b.Flush()

	ident := newIndentation(config.useIndentation, b)
	if config.printSelf && n.Type != DocumentNode {
		err = outputXML(b, n, preserveSpaces, config, ident)
	} else {
		for n := n.FirstChild; n != nil; n = n.NextSibling {
			err = outputXML(b, n, preserveSpaces, config, ident)
			if err != nil {
				break
			}
		}
	}
	return
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

// AddSibling adds a new node 'n' as a last node of sibling chain for a given node 'sibling'.
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

// AddImmediateSibling adds a new node 'n' as immediate sibling a given node 'sibling'.
func AddImmediateSibling(sibling, n *Node) {
	n.Parent = sibling.Parent
	n.NextSibling = sibling.NextSibling
	sibling.NextSibling = n
	n.PrevSibling = sibling
	if n.NextSibling != nil {
		n.NextSibling.PrevSibling = n
	} else if n.Parent != nil {
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

// GetRoot returns a root of the tree where 'n' is a node.
func GetRoot(n *Node) *Node {
	if n == nil {
		return nil
	}
	root := n
	for root.Parent != nil {
		root = root.Parent
	}
	return root
}
