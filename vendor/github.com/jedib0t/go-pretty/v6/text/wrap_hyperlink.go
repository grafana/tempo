package text

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// hyperlinkWrapper keeps OSC 8 links separate from SGR formatting. In
// particular, OSC's command number 8 is not the SGR conceal attribute.
type hyperlinkWrapper struct {
	out     strings.Builder
	colors  EscSeqParser
	link    string
	linkEnd string
	lineLen int
	width   int
}

func wrapHyperlinks(str string, width int, soft, keepNewlines bool) string {
	str = expandHyperlinkTabs(str)
	if hyperlinkWordWidth(str) <= width {
		return str
	}
	w := hyperlinkWrapper{width: width}
	separator := "\n\n"
	if keepNewlines {
		separator = "\n"
	}
	for paragraphIdx, paragraph := range hyperlinkParagraphs(str, separator) {
		if paragraphIdx > 0 {
			for range separator {
				w.newline(false)
			}
		}
		w.paragraph(paragraph, soft)
	}
	w.closeFormatting()
	return w.out.String()
}

func (w *hyperlinkWrapper) paragraph(paragraph string, soft bool) {
	words := hyperlinkWords(paragraph)
	for idx, word := range words {
		spacing := 0
		if w.lineLen > 0 {
			spacing = 1
		}
		if soft && w.lineLen > 0 && w.lineLen+spacing+hyperlinkWordWidth(word) > w.width {
			w.newline(true)
		} else if spacing > 0 {
			w.out.WriteByte(' ')
			w.lineLen++
		}
		w.word(word)
		if w.lineLen >= w.width && idx < len(words)-1 {
			w.newline(true)
		}
	}
}

func expandHyperlinkTabs(str string) string {
	if !strings.ContainsRune(str, '\t') {
		return str
	}
	var out strings.Builder
	for len(str) > 0 {
		token, rest, escape := hyperlinkToken(str)
		str = rest
		if !escape && token == "\t" {
			out.WriteString("    ")
		} else {
			out.WriteString(token)
		}
	}
	return out.String()
}

func (w *hyperlinkWrapper) word(word string) {
	for len(word) > 0 {
		token, rest, escape := hyperlinkToken(word)
		if !escape {
			// text is written out a grapheme cluster at a time, so that a
			// cluster is not split between two lines
			size, clusterWidth := wrapCluster(word, w.width)
			if w.overflows(clusterWidth) {
				w.newline(false)
			}
			w.out.WriteString(word[:size])
			w.lineLen += clusterWidth
			word = word[size:]
			continue
		}
		word = rest

		// A new link after a full line belongs to the following line. A closing
		// link is consumed first so the following text stays outside that link.
		if strings.HasPrefix(token, escapeStartConcealOSI) {
			uri, end, complete := hyperlinkTarget(token)
			if complete {
				if uri != "" && w.overflows(hyperlinkClusterWidth(word, w.width)) {
					w.newline(false)
				}
				if uri == "" {
					w.link, w.linkEnd = "", ""
				} else {
					w.link, w.linkEnd = token, "\x1b]8;;"+end
				}
			}
		} else {
			w.colors.ParseString(token)
		}
		w.out.WriteString(token)
	}
}

// overflows returns true if a grapheme cluster of the given width does not fit
// in what is left of the line. One that is wider than a whole line can never
// fit, so it is left to overflow a line of its own; and a zero-width one always
// fits.
func (w *hyperlinkWrapper) overflows(clusterWidth int) bool {
	return clusterWidth > 0 && w.lineLen > 0 && w.lineLen+clusterWidth > w.width
}

func (w *hyperlinkWrapper) closeFormatting() {
	if w.link != "" {
		w.out.WriteString(w.linkEnd)
	}
	if w.colors.IsOpen() {
		w.out.WriteString(EscapeReset)
	}
}

func (w *hyperlinkWrapper) newline(pad bool) {
	w.closeFormatting()
	if pad && w.lineLen < w.width {
		w.out.WriteString(strings.Repeat(" ", w.width-w.lineLen))
	}
	w.out.WriteByte('\n')
	w.out.WriteString(w.colors.Sequence())
	w.out.WriteString(w.link)
	w.lineLen = 0
}

func hyperlinkParagraphs(str, separator string) []string {
	var paragraphs []string
	var paragraph strings.Builder
	for len(str) > 0 {
		token, rest, escape := hyperlinkToken(str)
		if !escape && strings.HasPrefix(str, separator) {
			paragraphs = append(paragraphs, paragraph.String())
			paragraph.Reset()
			str = str[len(separator):]
		} else {
			paragraph.WriteString(token)
			str = rest
		}
	}
	return append(paragraphs, paragraph.String())
}

func hyperlinkTarget(token string) (uri, end string, complete bool) {
	switch {
	case strings.HasSuffix(token, "\x1b\\"):
		end = "\x1b\\"
	case strings.HasSuffix(token, "\a"):
		end = "\a"
	default:
		return "", "", false
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(token, escapeStartConcealOSI), end)
	_, uri, complete = strings.Cut(payload, ";")
	return
}

// Keep escape sequences intact while finding word boundaries, including spaces
// in OSC parameters. Only visible whitespace separates words.
func hyperlinkWords(str string) []string {
	var words []string
	var word strings.Builder
	visible := false
	for len(str) > 0 {
		token, rest, escape := hyperlinkToken(str)
		str = rest
		r, _ := utf8.DecodeRuneInString(token)
		if !escape && unicode.IsSpace(r) {
			if visible {
				words = append(words, word.String())
				word.Reset()
				visible = false
			}
		} else {
			word.WriteString(token)
			visible = visible || !escape
		}
	}
	if word.Len() > 0 {
		if !visible && len(words) > 0 {
			words[len(words)-1] += word.String()
		} else {
			words = append(words, word.String())
		}
	}
	return words
}

func hyperlinkWordWidth(word string) int {
	width := 0
	for len(word) > 0 {
		token, rest, escape := hyperlinkToken(word)
		word = rest
		if !escape {
			r, _ := utf8.DecodeRuneInString(token)
			width += RuneWidth(r)
		}
	}
	return width
}

// hyperlinkClusterWidth returns the width of the first visible grapheme cluster
// in word, or 0 if there is none.
func hyperlinkClusterWidth(word string, wrapLen int) int {
	for len(word) > 0 {
		_, rest, escape := hyperlinkToken(word)
		if !escape {
			_, clusterWidth := wrapCluster(word, wrapLen)
			return clusterWidth
		}
		word = rest
	}
	return 0
}

func hyperlinkToken(str string) (token, rest string, escape bool) {
	if str[0] != byte(EscapeStartRune) {
		_, size := utf8.DecodeRuneInString(str)
		return str[:size], str[size:], false
	}
	// The parser also resets its state after its size limit. Find actual OSC 8
	// terminators here so a reset cannot split a URL into visible text.
	if strings.HasPrefix(str, escapeStartConcealOSI) {
		for idx := len(escapeStartConcealOSI); idx < len(str); idx++ {
			if str[idx] == '\a' {
				return str[:idx+1], str[idx+1:], true
			}
			if str[idx] == byte(EscapeStartRune) && idx+1 < len(str) && str[idx+1] == '\\' {
				return str[:idx+2], str[idx+2:], true
			}
		}
		return str, "", true
	}
	parser := EscSeqParser{skipParse: true}
	for idx, r := range str {
		parser.Consume(r)
		if !parser.InSequence() {
			_, size := utf8.DecodeRuneInString(str[idx:])
			end := idx + size
			return str[:end], str[end:], true
		}
	}
	return str, "", true
}
