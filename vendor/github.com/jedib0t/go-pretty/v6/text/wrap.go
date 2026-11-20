package text

import (
	"strings"
	"unicode/utf8"
)

// WrapHard wraps a string to the given length using a newline. Handles strings
// with ANSI escape sequences (such as text color) without breaking the text
// formatting. Breaks all words that go beyond the line boundary.
//
// For examples, refer to the unit-tests or GoDoc examples.
func WrapHard(str string, wrapLen int) string {
	if wrapLen <= 0 {
		return ""
	}
	if strings.Contains(str, escapeStartConcealOSI) {
		return wrapHyperlinks(str, wrapLen, false, false)
	}
	str = strings.ReplaceAll(str, "\t", "    ")
	sLen := StringWidthWithoutEscSequences(str)
	if sLen <= wrapLen {
		return str
	}

	out := &strings.Builder{}
	out.Grow(sLen + (sLen / wrapLen))
	for idx, paragraph := range strings.Split(str, "\n\n") {
		if idx > 0 {
			out.WriteString("\n\n")
		}
		wrapHard(paragraph, wrapLen, out)
	}

	return out.String()
}

// WrapSoft wraps a string to the given length using a newline. Handles strings
// with ANSI escape sequences (such as text color) without breaking the text
// formatting. Tries to move words that go beyond the line boundary to the next
// line.
//
// For examples, refer to the unit-tests or GoDoc examples.
func WrapSoft(str string, wrapLen int) string {
	if wrapLen <= 0 {
		return ""
	}
	if strings.Contains(str, escapeStartConcealOSI) {
		return wrapHyperlinks(str, wrapLen, true, false)
	}
	str = strings.ReplaceAll(str, "\t", "    ")
	sLen := StringWidthWithoutEscSequences(str)
	if sLen <= wrapLen {
		return str
	}

	out := &strings.Builder{}
	out.Grow(sLen + (sLen / wrapLen))
	for idx, paragraph := range strings.Split(str, "\n\n") {
		if idx > 0 {
			out.WriteString("\n\n")
		}
		wrapSoft(paragraph, wrapLen, out)
	}

	return out.String()
}

// WrapText is very similar to WrapHard except for one minor difference. Unlike
// WrapHard which discards line-breaks and respects only paragraph-breaks, this
// function respects line-breaks too.
//
// For examples, refer to the unit-tests or GoDoc examples.
func WrapText(str string, wrapLen int) string {
	if wrapLen <= 0 {
		return ""
	}
	if strings.Contains(str, escapeStartConcealOSI) {
		return wrapHyperlinks(str, wrapLen, false, true)
	}
	str = strings.ReplaceAll(str, "\t", "    ")
	sLen := StringWidthWithoutEscSequences(str)
	if sLen <= wrapLen {
		return str
	}

	out := &strings.Builder{}
	out.Grow(sLen + (sLen / wrapLen))
	for idx, line := range strings.Split(str, "\n") {
		if idx > 0 {
			out.WriteString("\n")
		}
		wrapHard(line, wrapLen, out)
	}

	return out.String()
}

// appendCluster writes a grapheme cluster out, on a new line if it does not fit
// in what is left of the current one.
func appendCluster(cluster string, clusterWidth int, wrapLen int, lineLen *int, lastSeenEscSeq string, out *strings.Builder) {
	// a cluster that would overflow the line is moved to the next line, unless
	// the line is empty (it can never fit, so it is placed on a line of its
	// own)
	if *lineLen > 0 && *lineLen+clusterWidth > wrapLen {
		if lastSeenEscSeq != "" {
			// terminate escape sequence and the line; and restart the escape
			// sequence in the next line
			out.WriteString(EscapeReset)
			out.WriteRune('\n')
			out.WriteString(lastSeenEscSeq)
		} else {
			// just start a new line
			out.WriteRune('\n')
		}
		// reset line index to 0th character
		*lineLen = 0
	}

	for _, char := range cluster {
		out.WriteRune(char)
	}
	*lineLen += clusterWidth
}

func appendWord(word string, lineIdx *int, lastSeenEscSeq string, wrapLen int, out *strings.Builder) {
	esp, escSeq := EscSeqParser{skipParse: true}, ""
	for idx := 0; idx < len(word); {
		// escape sequences are written out as they are, a rune at a time
		if esp.InSequence() || word[idx] == byte(EscapeStartRune) {
			char, size := utf8.DecodeRuneInString(word[idx:])
			idx += size
			if !esp.InSequence() {
				escSeq = ""
			}
			esp.Consume(char)
			if len(escSeq) < escSeqMaxLength {
				escSeq += string(char)
			}
			out.WriteRune(char)

			// A sequence that sets colors or attributes remains in force for
			// the text after it, and has to be restarted on every new line.
			// Any other sequence (moving the cursor, erasing the line, ...) is
			// done once it has been written out.
			if !esp.InSequence() && strings.HasPrefix(escSeq, EscapeStartCSI) && strings.HasSuffix(escSeq, EscapeStopCSI) {
				lastSeenEscSeq = escSeq
				if lastSeenEscSeq == EscapeReset {
					lastSeenEscSeq = ""
				}
			}
			continue
		}

		// text is written out a grapheme cluster at a time, so that a cluster
		// (like an emoji with a skin tone) is not split between two lines
		size, clusterWidth := 1, 0
		if idx+1 < len(word) && word[idx] < utf8.RuneSelf && word[idx+1] < utf8.RuneSelf {
			// hot path: an ASCII character followed by another one is always
			// a cluster of its own; see isStandaloneRune
			clusterWidth = RuneWidth(rune(word[idx]))
		} else {
			size, clusterWidth = wrapCluster(word[idx:], wrapLen)
		}
		appendCluster(word[idx:idx+size], clusterWidth, wrapLen, lineIdx, lastSeenEscSeq, out)
		idx += size
	}
}

// wrapCluster returns the size in bytes and the display width of the grapheme
// cluster at the beginning of str, which must not be empty. A cluster that is
// wider than a whole line has to be split to keep to wrapLen; all that is
// returned of such a cluster is its first rune.
func wrapCluster(str string, wrapLen int) (int, int) {
	size, clusterWidth := trimCluster(str)
	if clusterWidth > wrapLen {
		char, charSize := utf8.DecodeRuneInString(str)
		return charSize, RuneWidth(char)
	}
	return size, clusterWidth
}

func terminateLine(wrapLen int, lineLen *int, lastSeenEscSeq string, out *strings.Builder) {
	if *lineLen < wrapLen {
		out.WriteString(strings.Repeat(" ", wrapLen-*lineLen))
	}
	// something is already on the line; terminate it
	if lastSeenEscSeq != "" {
		out.WriteString(EscapeReset)
	}
	out.WriteRune('\n')
	out.WriteString(lastSeenEscSeq)
	*lineLen = 0
}

func terminateOutput(lastSeenEscSeq string, out *strings.Builder) {
	if lastSeenEscSeq != "" && lastSeenEscSeq != EscapeReset && !strings.HasSuffix(out.String(), EscapeReset) {
		out.WriteString(EscapeReset)
	}
}

func wrapHard(paragraph string, wrapLen int, out *strings.Builder) {
	esp := EscSeqParser{}
	lineLen, lastSeenEscSeq := 0, ""
	words := strings.Fields(paragraph)
	for wordIdx, word := range words {
		if openEscSeq := esp.ParseString(word); openEscSeq != "" {
			lastSeenEscSeq = openEscSeq
		}
		if lineLen > 0 {
			out.WriteRune(' ')
			lineLen++
		}

		wordLen := StringWidthWithoutEscSequences(word)
		if lineLen+wordLen <= wrapLen { // word fits within the line
			out.WriteString(word)
			lineLen += wordLen
		} else { // word doesn't fit within the line; hard-wrap
			appendWord(word, &lineLen, lastSeenEscSeq, wrapLen, out)
		}

		// end of line; but more words incoming
		if lineLen >= wrapLen && wordIdx < len(words)-1 {
			terminateLine(wrapLen, &lineLen, lastSeenEscSeq, out)
		}
	}
	terminateOutput(lastSeenEscSeq, out)
}

func wrapSoft(paragraph string, wrapLen int, out *strings.Builder) {
	esp := EscSeqParser{}
	lineLen, lastSeenEscSeq := 0, ""
	words := strings.Fields(paragraph)
	for wordIdx, word := range words {
		if openEscSeq := esp.ParseString(word); openEscSeq != "" {
			lastSeenEscSeq = openEscSeq
		}

		spacing, spacingLen := wrapSoftSpacing(lineLen)
		wordLen := StringWidthWithoutEscSequences(word)
		if lineLen+spacingLen+wordLen <= wrapLen { // word fits within the line
			out.WriteString(spacing)
			out.WriteString(word)
			lineLen += spacingLen + wordLen
		} else { // word doesn't fit within the line
			lineLen = wrapSoftLastWordInLine(wrapLen, lineLen, lastSeenEscSeq, wordLen, word, out)
		}

		// end of line; but more words incoming
		if lineLen >= wrapLen && wordIdx < len(words)-1 {
			terminateLine(wrapLen, &lineLen, lastSeenEscSeq, out)
		}
	}
	terminateOutput(lastSeenEscSeq, out)
}

func wrapSoftLastWordInLine(wrapLen int, lineLen int, lastSeenEscSeq string, wordLen int, word string, out *strings.Builder) int {
	if lineLen > 0 { // something is already on the line; terminate it
		terminateLine(wrapLen, &lineLen, lastSeenEscSeq, out)
	}
	if wordLen <= wrapLen { // word fits within a single line
		out.WriteString(word)
		lineLen = wordLen
	} else { // word doesn't fit within a single line; hard-wrap
		appendWord(word, &lineLen, lastSeenEscSeq, wrapLen, out)
	}
	return lineLen
}

func wrapSoftSpacing(lineLen int) (string, int) {
	spacing, spacingLen := "", 0
	if lineLen > 0 {
		spacing, spacingLen = " ", 1
	}
	return spacing, spacingLen
}
