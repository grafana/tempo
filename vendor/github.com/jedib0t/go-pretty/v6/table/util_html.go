package table

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
)

// convertEscSequencesToSpans converts ANSI escape sequences to HTML <span> tags
// with CSS classes; 24-bit colors have no class, and become inline styles.
func convertEscSequencesToSpans(str string) string {
	converter := newEscSeqToSpanConverter()
	return converter.Convert(str)
}

// escSeqToSpanConverter converts ANSI escape sequences to HTML <span> tags with CSS classes.
type escSeqToSpanConverter struct {
	result strings.Builder
	esp    text.EscSeqParser
	// currentAttrs has the attributes of the <span> that is open; empty if
	// none is.
	currentAttrs string
}

// newEscSeqToSpanConverter creates a new escape sequence to span converter.
func newEscSeqToSpanConverter() *escSeqToSpanConverter {
	return &escSeqToSpanConverter{}
}

// Convert converts ANSI escape sequences in the string to HTML <span> tags with CSS classes.
func (c *escSeqToSpanConverter) Convert(str string) string {
	c.reset()

	// Process the string character by character
	for _, char := range str {
		if !c.esp.ParseRune(char) {
			// Regular character, escape it for HTML safety and write it
			// (will be inside current span if colors are active)
			c.writeEscapedRune(char)
		} else if !c.esp.InSequence() {
			// We just finished an escape sequence, update the span
			c.updateSpan(c.spanAttributes())
		}
	}

	c.updateSpan("")

	return c.result.String()
}

// cssClasses returns the CSS class names for the colors and attributes that
// are active.
func (c *escSeqToSpanConverter) cssClasses() string {
	codes := c.esp.Codes()
	colors := make(text.Colors, len(codes))
	for idx, code := range codes {
		colors[idx] = text.Color(code)
	}
	return colors.CSSClasses()
}

// cssStyle returns the CSS declarations for the 24-bit colors that are active.
// There are too many of these for a stylesheet to have a class for each, so
// the color itself is used.
func (c *escSeqToSpanConverter) cssStyle() string {
	var style []string
	if r, g, b, ok := c.esp.FgRGB(); ok {
		style = append(style, fmt.Sprintf("color: #%02x%02x%02x;", r, g, b))
	}
	if r, g, b, ok := c.esp.BgRGB(); ok {
		style = append(style, fmt.Sprintf("background-color: #%02x%02x%02x;", r, g, b))
	}
	return strings.Join(style, " ")
}

// reset initializes the converter state for a new conversion.
func (c *escSeqToSpanConverter) reset() {
	c.result.Reset()
	c.esp = text.EscSeqParser{}
	c.currentAttrs = ""
}

// spanAttributes returns the attributes of the <span> that the text needs for
// the colors and attributes that are active; empty if it needs no <span>.
func (c *escSeqToSpanConverter) spanAttributes() string {
	attrs := ""
	if class := c.cssClasses(); class != "" {
		attrs += " class=\"" + class + "\""
	}
	if style := c.cssStyle(); style != "" {
		attrs += " style=\"" + style + "\""
	}
	return attrs
}

// updateSpan closes the open span and opens a new one, if the attributes of
// the span that the text needs have changed. The attributes are compared
// instead of the codes because a code without a CSS class makes no difference
// to the HTML.
func (c *escSeqToSpanConverter) updateSpan(attrs string) {
	if attrs == c.currentAttrs {
		return
	}

	if c.currentAttrs != "" {
		c.result.WriteString("</span>")
	}
	if attrs != "" {
		c.result.WriteString("<span")
		c.result.WriteString(attrs)
		c.result.WriteString(">")
	}
	c.currentAttrs = attrs
}

// writeEscapedRune writes a rune to the result, escaping it if necessary for HTML safety.
func (c *escSeqToSpanConverter) writeEscapedRune(char rune) {
	switch char {
	case '<':
		c.result.WriteString("&lt;")
	case '>':
		c.result.WriteString("&gt;")
	case '&':
		c.result.WriteString("&amp;")
	case '"':
		c.result.WriteString("&#34;")
	case '\'':
		c.result.WriteString("&#39;")
	default:
		// Most characters don't need escaping, write directly
		c.result.WriteRune(char)
	}
}
