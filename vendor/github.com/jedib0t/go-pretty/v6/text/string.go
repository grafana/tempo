package text

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
	"golang.org/x/text/width"
)

// RuneWidth stuff
var (
	rwCondition = runewidth.NewCondition()
)

// InsertEveryN inserts the rune every N characters in the string. For ex.:
//
//	InsertEveryN("Ghost", '-', 1) == "G-h-o-s-t"
//	InsertEveryN("Ghost", '-', 2) == "Gh-os-t"
//	InsertEveryN("Ghost", '-', 3) == "Gho-st"
//	InsertEveryN("Ghost", '-', 4) == "Ghos-t"
//	InsertEveryN("Ghost", '-', 5) == "Ghost"
//
// The count restarts on every line, and the rune is never inserted in the
// middle of a grapheme cluster (like an emoji built with zero-width joiners),
// or in the middle of a wide character; a wide character that crosses the
// N-th column gets the rune inserted after it.
func InsertEveryN(str string, runeToInsert rune, n int) string {
	if n <= 0 {
		return str
	}

	var out strings.Builder
	out.Grow(len(str) + utf8.RuneLen(runeToInsert)*(len(str)/n))
	lineLen, esp, state := 0, EscSeqParser{skipParse: true}, -1
	for len(str) > 0 {
		var cluster string
		cluster, str, _, state = uniseg.StepString(str, state)
		for idx, c := range cluster {
			if esp.InSequence() {
				esp.Consume(c)
				out.WriteRune(c)
				continue
			}
			esp.Consume(c)
			if esp.InSequence() {
				out.WriteRune(c)
				continue
			}

			// c starts the visible part of the cluster; ESC is a control
			// character, which always forms a cluster of its own, so no
			// escape sequence can start further inside this cluster
			visible := cluster[idx:]
			if strings.ContainsAny(visible, "\r\n") {
				lineLen = 0
			} else if vWidth := StringWidthWithoutEscSequences(visible); vWidth > 0 {
				if lineLen >= n {
					out.WriteRune(runeToInsert)
					lineLen = 0
				}
				lineLen += vWidth
			}
			out.WriteString(visible)
			break
		}
	}
	return out.String()
}

// LongestLineLen returns the length of the longest "line" within the
// argument string. For ex.:
//
//	LongestLineLen("Ghost!\nCome back here!\nRight now!") == 15
func LongestLineLen(str string) int {
	maxLength, currLength, esp := 0, 0, EscSeqParser{skipParse: true}
	//fmt.Println(str)
	for _, c := range str {
		//fmt.Printf("%03d | %03d | %c | %5v | %v | %#v\n", idx, c, c, esp.inEscSeq, esp.Codes(), esp.escapeSeq)
		if esp.InSequence() {
			esp.Consume(c)
			continue
		}
		esp.Consume(c)
		if c == '\n' {
			if currLength > maxLength {
				maxLength = currLength
			}
			currLength = 0
		} else if !esp.InSequence() {
			currLength += RuneWidth(c)
		}
	}
	if currLength > maxLength {
		maxLength = currLength
	}
	return maxLength
}

// OverrideRuneWidthEastAsianWidth overrides the East Asian width detection in
// the runewidth library. This is primarily for advanced use cases.
//
// Box drawing (U+2500-U+257F) and block element (U+2580-U+259F) characters
// are automatically handled and always reported as width 1, regardless of
// this setting, fixing alignment issues that previously required setting this
// to false.
//
// Setting this to false forces runewidth to treat all characters as if in an
// English locale. Warning: this may cause East Asian characters (Chinese,
// Japanese, Korean) to be incorrectly reported as width 1 instead of 2.
//
// See:
// * https://github.com/mattn/go-runewidth/issues/64
// * https://github.com/jedib0t/go-pretty/issues/220
// * https://github.com/jedib0t/go-pretty/issues/204
func OverrideRuneWidthEastAsianWidth(val bool) {
	rwCondition.EastAsianWidth = val
}

// Pad pads the given string with as many characters as needed to make it as
// long as specified (maxLen). This function does not count escape sequences
// while calculating length of the string. Ex.:
//
//	Pad("Ghost", 0, ' ') == "Ghost"
//	Pad("Ghost", 3, ' ') == "Ghost"
//	Pad("Ghost", 5, ' ') == "Ghost"
//	Pad("Ghost", 7, ' ') == "Ghost  "
//	Pad("Ghost", 10, '.') == "Ghost....."
func Pad(str string, maxLen int, paddingChar rune) string {
	strLen := StringWidthWithoutEscSequences(str)
	if strLen < maxLen {
		str += strings.Repeat(string(paddingChar), maxLen-strLen)
	}
	return str
}

// ProcessCRLF converts "\r\n" to "\n", and processes lone "\r" by moving the
// cursor/carriage to the start of the line and overwrites the contents
// accordingly. Ex.:
//
// ProcessCRLF("abc") == "abc"
// ProcessCRLF("abc\r\ndef") == "abc\ndef"
// ProcessCRLF("abc\r\ndef\rghi") == "abc\nghi"
// ProcessCRLF("abc\r\ndef\rghi\njkl") == "abc\nghi\njkl"
// ProcessCRLF("abc\r\ndef\rghi\njkl\r") == "abc\nghi\njkl"
// ProcessCRLF("abc\r\ndef\rghi\rjkl\rmn") == "abc\nmnl"
func ProcessCRLF(str string) string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	if !strings.Contains(str, "\r") {
		return str
	}

	lines := strings.Split(str, "\n")
	for lineIdx, line := range lines {
		if !strings.Contains(line, "\r") {
			continue
		}

		lineRunes, newLineRunes := []rune(line), make([]rune, 0)
		for idx, realIdx := 0, 0; idx < len(lineRunes); idx++ {
			// if a CR, move "cursor" back to beginning of line
			if lineRunes[idx] == '\r' {
				realIdx = 0
				continue
			}

			// if cursor is not at end, overwrite
			if realIdx < len(newLineRunes) {
				newLineRunes[realIdx] = lineRunes[idx]
			} else { // else append
				newLineRunes = append(newLineRunes, lineRunes[idx])
			}
			realIdx++
		}
		lines[lineIdx] = string(newLineRunes)
	}
	return strings.Join(lines, "\n")
}

// RepeatAndTrim repeats the given string until it is maxRunes display columns
// wide. Escape sequences are ignored while measuring, and the ones found before
// and after the text are used just once, around the repeated text. The result
// can be narrower than maxRunes if a wide character does not fit in the end.
// For ex.:
//
//	RepeatAndTrim("", 5) == ""
//	RepeatAndTrim("Ghost", 0) == ""
//	RepeatAndTrim("Ghost", 5) == "Ghost"
//	RepeatAndTrim("Ghost", 7) == "GhostGh"
//	RepeatAndTrim("Ghost", 10) == "GhostGhost"
//	RepeatAndTrim("\x1b[33mGhost\x1b[0m", 7) == "\x1b[33mGhostGh\x1b[0m"
//	RepeatAndTrim("生命", 5) == "生命"
func RepeatAndTrim(str string, maxRunes int) string {
	if str == "" || maxRunes <= 0 {
		return ""
	}
	if !strings.ContainsRune(str, EscapeStartRune) {
		return repeatAndTrim(str, maxRunes)
	}

	// split the string into the escape sequences leading up to the text, the
	// text, and the escape sequences trailing it
	textBegin, textEnd, esp := -1, 0, EscSeqParser{skipParse: true}
	for idx := 0; idx < len(str); {
		c, size := utf8.DecodeRuneInString(str[idx:])
		idx += size
		if esp.InSequence() || c == EscapeStartRune {
			esp.Consume(c)
			continue
		}
		if textBegin < 0 {
			textBegin = idx - size
		}
		textEnd = idx
	}
	if textBegin < 0 { // nothing but escape sequences
		return str
	}
	prefix, text, suffix := str[:textBegin], str[textBegin:textEnd], str[textEnd:]
	if strings.ContainsRune(text, EscapeStartRune) {
		// escape sequences within the text change how the text after them is
		// rendered, so every copy needs the leading escape sequences again
		prefix, text = "", str[:textEnd]
	}
	return prefix + repeatAndTrim(text, maxRunes) + suffix
}

// repeatAndTrim repeats the given string until it is maxLen display columns
// wide, with its escape sequences repeated along with it.
func repeatAndTrim(str string, maxLen int) string {
	strWidth := StringWidthWithoutEscSequences(str)
	if strWidth == 0 || strWidth == maxLen {
		return str
	}
	if maxLen%strWidth == 0 {
		// whole copies add up to the width exactly; there is nothing to trim
		return strings.Repeat(str, maxLen/strWidth)
	}
	return Trim(strings.Repeat(str, maxLen/strWidth+1), maxLen)
}

// RuneCount is similar to utf8.RuneCountInString, except for the fact that it
// ignores escape sequences while counting. For ex.:
//
//	RuneCount("") == 0
//	RuneCount("Ghost") == 5
//	RuneCount("\x1b[33mGhost\x1b[0m") == 5
//	RuneCount("\x1b[33mGhost\x1b[0") == 5
//
// Deprecated: in favor of RuneWidthWithoutEscSequences
func RuneCount(str string) int {
	return StringWidthWithoutEscSequences(str)
}

// RuneWidth returns the display width of a rune. Width accuracy depends on
// the terminal font, as character width is font-dependent. Examples:
//
//	RuneWidth('A') == 1
//	RuneWidth('ツ') == 2
//	RuneWidth('⊙') == 1
//	RuneWidth('︿') == 2
//	RuneWidth(0x27) == 0
//
// Box drawing (U+2500-U+257F) and block element (U+2580-U+259F) characters
// are always treated as width 1, regardless of locale, to ensure proper
// alignment in tables and progress indicators. This fixes incorrect width 2
// reporting in East Asian locales (e.g., LANG=zh_CN.UTF-8).
//
// See:
// * https://github.com/mattn/go-runewidth/issues/64
// * https://github.com/jedib0t/go-pretty/issues/220
// * https://github.com/jedib0t/go-pretty/issues/204
func RuneWidth(r rune) int {
	// ASCII is the common case, and needs none of the lookups below: printable
	// characters are one column wide, control characters are zero.
	if r < utf8.RuneSelf {
		if r >= 0x20 && r < 0x7f {
			return 1
		}
		return 0
	}
	if isSingleCellRune(r) {
		return 1
	}
	return rwCondition.RuneWidth(r)
}

// isSingleCellRune returns true for box drawing (U+2500-U+257F) and block
// element (U+2580-U+259F) characters, which are always treated as width 1
// regardless of locale.
func isSingleCellRune(r rune) bool {
	return (r >= 0x2500 && r <= 0x257F) || (r >= 0x2580 && r <= 0x259F)
}

// RuneWidthWithoutEscSequences is similar to RuneWidth, except for the fact
// that it ignores escape sequences while counting. For ex.:
//
//	RuneWidthWithoutEscSequences("") == 0
//	RuneWidthWithoutEscSequences("Ghost") == 5
//	RuneWidthWithoutEscSequences("\x1b[33mGhost\x1b[0m") == 5
//	RuneWidthWithoutEscSequences("\x1b[33mGhost\x1b[0") == 5
//
// deprecated: use StringWidthWithoutEscSequences instead
func RuneWidthWithoutEscSequences(str string) int {
	return StringWidthWithoutEscSequences(str)
}

// Snip returns the given string with a fixed length. For ex.:
//
//	Snip("Ghost", 0, "~") == "Ghost"
//	Snip("Ghost", 1, "~") == "~"
//	Snip("Ghost", 3, "~") == "Gh~"
//	Snip("Ghost", 5, "~") == "Ghost"
//	Snip("Ghost", 7, "~") == "Ghost  "
//	Snip("\x1b[33mGhost\x1b[0m", 7, "~") == "\x1b[33mGhost\x1b[0m  "
func Snip(str string, length int, snipIndicator string) string {
	if length > 0 {
		lenStr := StringWidthWithoutEscSequences(str)
		if lenStr > length {
			snipIndicator = Trim(snipIndicator, length)
			lenStrFinal := length - StringWidthWithoutEscSequences(snipIndicator)
			return Trim(str, lenStrFinal) + snipIndicator
		}
	}
	return str
}

// StringWidth is similar to RuneWidth, except it works on a string. For
// ex.:
//
//	StringWidth("Ghost 生命"): 10
//	StringWidth("\x1b[33mGhost 生命\x1b[0m"): 19
//	StringWidth("┌──┐"): 4
func StringWidth(str string) (width int) {
	// Counted the way runewidth counts, one grapheme cluster at a time taking
	// the first rune with a width, but through RuneWidth so that the box
	// drawing and block element override applies here too.
	g := uniseg.NewGraphemes(str)
	for g.Next() {
		for _, r := range g.Runes() {
			if w := RuneWidth(r); w > 0 {
				width += w
				break
			}
		}
	}
	return width
}

// StringWidthWithoutEscSequences is similar to RuneWidth, except for the fact
// that it ignores escape sequences while counting. For ex.:
//
//	StringWidthWithoutEscSequences("") == 0
//	StringWidthWithoutEscSequences("Ghost") == 5
//	StringWidthWithoutEscSequences("\x1b[33mGhost\x1b[0m") == 5
//	StringWidthWithoutEscSequences("\x1b[33mGhost\x1b[0") == 5
//	StringWidthWithoutEscSequences("Ghost 生命"): 10
//	StringWidthWithoutEscSequences("\x1b[33mGhost 生命\x1b[0m"): 10
func StringWidthWithoutEscSequences(str string) int {
	// fast-path the common case of a string without escape sequences
	if !strings.ContainsRune(str, EscapeStartRune) {
		count := 0
		for _, c := range str {
			count += RuneWidth(c)
		}
		return count
	}

	count, esp := 0, EscSeqParser{skipParse: true}
	for _, c := range str {
		if esp.InSequence() {
			esp.Consume(c)
			continue
		}
		esp.Consume(c)
		if !esp.InSequence() {
			count += RuneWidth(c)
		}
	}
	return count
}

// Trim trims a string to the given display width (maxLen columns) while
// ignoring escape sequences. Wide East Asian characters count as two columns,
// so the result never exceeds maxLen columns, and a grapheme cluster is never
// cut in the middle. For ex.:
//
//	Trim("Ghost", 3) == "Gho"
//	Trim("Ghost", 6) == "Ghost"
//	Trim("\x1b[33mGhost\x1b[0m", 3) == "\x1b[33mGho\x1b[0m"
//	Trim("\x1b[33mGhost\x1b[0m", 6) == "\x1b[33mGhost\x1b[0m"
//	Trim("生命生命", 3) == "生"
//	Trim("生命生命", 4) == "生命"
func Trim(str string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	var out strings.Builder
	out.Grow(maxLen)

	outLen, esp := 0, EscSeqParser{skipParse: true}
	for idx := 0; idx < len(str); {
		// Escape sequences are copied as they are. ESC is a control character,
		// which always forms a grapheme cluster of its own, so no escape
		// sequence can start inside a cluster.
		if esp.InSequence() || str[idx] == byte(EscapeStartRune) {
			sChr, size := utf8.DecodeRuneInString(str[idx:])
			esp.Consume(sChr)
			out.WriteRune(sChr)
			idx += size
			continue
		}

		// Count the display width of the whole grapheme cluster (wide East
		// Asian characters occupy two columns) instead of counting runes, so
		// the result never exceeds maxLen columns and never ends in the middle
		// of a cluster (like an emoji built with zero-width joiners).
		size, clusterWidth := 1, 0
		if idx+1 < len(str) && str[idx] < utf8.RuneSelf && str[idx+1] < utf8.RuneSelf {
			// hot path: an ASCII character followed by another one is always
			// a cluster of its own; see isStandaloneRune
			clusterWidth = RuneWidth(rune(str[idx]))
		} else {
			size, clusterWidth = trimCluster(str[idx:])
		}
		if outLen+clusterWidth > maxLen {
			// The cluster no longer fits: stop emitting visible text, but
			// keep copying any trailing escape sequences.
			return out.String() + trailingEscSequences(str[idx+size:], &esp)
		}
		outLen += clusterWidth
		out.WriteString(str[idx : idx+size])
		idx += size
	}
	return out.String()
}

// standaloneRuneRanges are the ranges of non-ASCII runes that isStandaloneRune
// returns true for.
var standaloneRuneRanges = [...][2]rune{
	{0x4E00, 0x9FFF}, // CJK Unified Ideographs
	{0x3041, 0x3096}, // Hiragana
	{0x30A1, 0x30FA}, // Katakana
	{0x3001, 0x3002}, // Ideographic comma and full stop
	{0xAC00, 0xD7A3}, // Hangul Syllables
	{0x2500, 0x259F}, // Box Drawing and Block Elements
	{0x00AE, 0x02FF}, // Latin-1 Supplement (after the soft hyphen) to Spacing Modifier Letters
	{0x00A0, 0x00AC}, // Latin-1 Supplement (before the soft hyphen)
	{0x0370, 0x0482}, // Greek and Cyrillic (before the combining marks)
	{0x048A, 0x052F}, // Cyrillic (after the combining marks) and its supplement
}

// isStandaloneRune returns true for the runes that form a grapheme cluster of
// their own whenever the rune that follows is a standalone rune too: ASCII and
// the runes in standaloneRuneRanges.
//
// "\r\n" is the one pair that is a single cluster instead; both are
// zero-width, so a width check can never separate them.
func isStandaloneRune(r rune) bool {
	if r < utf8.RuneSelf {
		return true
	}
	for _, runeRange := range standaloneRuneRanges {
		if r >= runeRange[0] && r <= runeRange[1] {
			return true
		}
	}
	return false
}

// trimCluster returns the size in bytes and the display width of the grapheme
// cluster at the beginning of str, which must not be empty.
func trimCluster(str string) (int, int) {
	sChr, size := rune(str[0]), 1
	if sChr >= utf8.RuneSelf {
		sChr, size = utf8.DecodeRuneInString(str)
	}
	// Segmenting text into grapheme clusters is slow; skip it for the common
	// case of a standalone rune followed by another standalone rune.
	if isStandaloneRune(sChr) {
		if size == len(str) || str[size] < utf8.RuneSelf {
			return size, RuneWidth(sChr)
		}
		if next, _ := utf8.DecodeRuneInString(str[size:]); isStandaloneRune(next) {
			return size, RuneWidth(sChr)
		}
	}

	cluster, _, _, _ := uniseg.FirstGraphemeClusterInString(str, -1)
	clusterWidth := 0
	for _, c := range cluster {
		clusterWidth += RuneWidth(c)
	}
	return len(cluster), clusterWidth
}

// trailingEscSequences returns only the escape sequences found in str, in
// order, dropping all the visible text. esp carries the state of the escape
// sequence parser from the text preceding str.
func trailingEscSequences(str string, esp *EscSeqParser) string {
	var out strings.Builder
	for _, c := range str {
		if esp.InSequence() {
			esp.Consume(c)
			out.WriteRune(c)
			continue
		}
		esp.Consume(c)
		if esp.InSequence() {
			out.WriteRune(c)
		}
	}
	return out.String()
}

// Widen is like width.Widen.String() but ignores escape sequences. For ex:
//
//	Widen("Ghost 生命"): "Ｇｈｏｓｔ\u3000生命"
//	Widen("\x1b[33mGhost 生命\x1b[0m"): "\x1b[33mＧｈｏｓｔ\u3000生命\x1b[0m"
func Widen(str string) string {
	sb := strings.Builder{}
	sb.Grow(len(str))

	esp := EscSeqParser{skipParse: true}
	for _, c := range str {
		if esp.InSequence() {
			sb.WriteRune(c)
			esp.Consume(c)
			continue
		}
		esp.Consume(c)
		if !esp.InSequence() {
			sb.WriteString(width.Widen.String(string(c)))
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}
