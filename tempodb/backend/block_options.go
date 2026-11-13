package backend

import (
	"slices"
	"strings"
)

// OptionsString renders the block options as "[key=value, key2=value2]", sorted by key so that
// it is independent of map iteration order. It is empty when there are no options. Blocks with
// different options must not be compacted together, so two option sets are equal exactly
// when their strings are equal: the characters that delimit the format are escaped with a backslash.
func (b *BlockMeta) OptionsString() string {
	if len(b.Options) == 0 {
		return ""
	}
	keys := make([]string, 0, len(b.Options))
	for k := range b.Options {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var sb strings.Builder
	sb.WriteByte('[')
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		_, _ = optionEscaper.WriteString(&sb, k)
		sb.WriteByte('=')
		_, _ = optionEscaper.WriteString(&sb, b.Options[k])
	}
	sb.WriteByte(']')
	return sb.String()
}

var optionEscaper = strings.NewReplacer(`\`, `\\`, `,`, `\,`, `=`, `\=`)
