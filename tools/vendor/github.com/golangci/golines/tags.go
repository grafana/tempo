package golines

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/dave/dst"
	"github.com/fatih/structtag"
)

var structTagRegexp = regexp.MustCompile("`([ ]*[a-zA-Z0-9_-]+:\".*\"[ ]*){2,}`")

// HasMultiKeyTags returns whether the given lines have a multikey struct line.
// It's used as an optimization step to avoid unnnecessary shortening rounds.
func HasMultiKeyTags(lines []string) bool {
	for _, line := range lines {
		if structTagRegexp.MatchString(line) {
			return true
		}
	}

	return false
}

// FormatStructTags formats struct tags so that the keys within each block of fields are aligned.
// It's not technically a shortening (and it usually makes these tags longer), so it's being
// kept separate from the core shortening logic for now.
//
// See the struct_tags fixture for examples.
func FormatStructTags(fieldList *dst.FieldList) {
	if fieldList == nil || len(fieldList.List) == 0 {
		return
	}

	blockFields := []*dst.Field{}

	// Divide fields into "blocks" so that we don't do alignments across blank lines
	for f, field := range fieldList.List {
		if f == 0 || field.Decorations().Before == dst.EmptyLine {
			alignTags(blockFields)
			blockFields = blockFields[:0]
		}

		blockFields = append(blockFields, field)
	}

	alignTags(blockFields)
}

// alignTags formats the struct tags within a single field block.
func alignTags(fields []*dst.Field) {
	if len(fields) == 0 {
		return
	}

	maxTagWidths := map[string]int{}
	tagKeys := []string{}
	tagKVs := make([]map[string]string, len(fields))

	maxTypeWidth := 0
	invalidWidths := false

	// First, scan over all field tags so that we can understand their values and widths
	for f, field := range fields {
		if len(field.Names) > 0 {
			typeWidth, err := getWidth(field.Type)
			if err != nil {
				// We couldn't figure out the proper width of this field
				invalidWidths = true
			} else if typeWidth > maxTypeWidth {
				maxTypeWidth = typeWidth
			}
		}

		if field.Tag == nil {
			continue
		}

		tagValue := field.Tag.Value

		// The dst library doesn't strip off the backticks, so we need to do this manually
		if tagValue[0] != '`' || tagValue[len(tagValue)-1] != '`' {
			continue
		}
		tagValue = tagValue[1 : len(tagValue)-1]

		subTags, err := structtag.Parse(tagValue)
		if err != nil {
			return
		}
		subTagKeys := subTags.Keys()

		structTag := reflect.StructTag(tagValue)

		for _, key := range subTagKeys {
			value := structTag.Get(key)

			// Tag is key, value, and some extra chars (two quotes + one colon)
			width := len(key) + tagValueLen(value) + 3

			if _, ok := maxTagWidths[key]; !ok {
				maxTagWidths[key] = width
				tagKeys = append(tagKeys, key)
			} else if width > maxTagWidths[key] {
				maxTagWidths[key] = width
			}

			if tagKVs[f] == nil {
				tagKVs[f] = map[string]string{}
			}

			tagKVs[f][key] = value
		}
	}

	// Go over all the fields again, replacing each tag with a reformatted one
	for f, field := range fields {
		if tagKVs[f] == nil {
			continue
		}

		tagComponents := []string{}

		if len(field.Names) == 0 && maxTypeWidth > 0 && !invalidWidths {
			// Add extra spacing at beginning so that tag aligns with named field tags
			tagComponents = append(tagComponents, "")

			for i := 0; i < maxTypeWidth; i++ {
				tagComponents[len(tagComponents)-1] += " "
			}
		}

		for _, key := range tagKeys {
			value, ok := tagKVs[f][key]
			lenUsed := 0

			if ok {
				tagComponents = append(tagComponents, fmt.Sprintf("%s:\"%s\"", key, value))
				lenUsed += len(key) + tagValueLen(value) + 3
			} else {
				tagComponents = append(tagComponents, "")
			}

			if len(field.Names) > 0 || !invalidWidths {
				lenRemaining := maxTagWidths[key] - lenUsed

				for i := 0; i < lenRemaining; i++ {
					tagComponents[len(tagComponents)-1] += " "
				}
			}
		}

		updatedTagValue := strings.TrimRight(strings.Join(tagComponents, " "), " ")
		field.Tag.Value = fmt.Sprintf("`%s`", updatedTagValue)
	}
}

// get real tag value's length, fix multi-byte character's length, such as `ï`
// or `中文`
func tagValueLen(s string) int {
	return len([]rune(s))
}

// getWidth tries to guess the formatted width of a dst node expression. If this isn't (yet)
// possible, it returns an error.
func getWidth(node dst.Node) (int, error) {
	switch n := node.(type) {
	case *dst.ArrayType:
		eltWidth, err := getWidth(n.Elt)
		if err != nil {
			return 0, err
		}

		return 2 + eltWidth, nil
	case *dst.ChanType:
		valWidth, err := getWidth(n.Value)
		if err != nil {
			return 0, err
		}

		isSend := n.Dir&dst.SEND > 0
		isRecv := n.Dir&dst.RECV > 0

		if isSend && isRecv {
			// Channel does not include an arrow
			return 5 + valWidth, nil
		}

		// Channel includes an arrow
		return 7 + valWidth, nil
	case *dst.Ident:
		return len(n.Name), nil
	case *dst.MapType:
		keyWidth, err := getWidth(n.Key)
		if err != nil {
			return 0, err
		}

		valWidth, err := getWidth(n.Value)
		if err != nil {
			return 0, err
		}

		return 5 + keyWidth + valWidth, nil
	case *dst.StarExpr:
		xWidth, err := getWidth(n.X)
		if err != nil {
			return 0, err
		}

		return 1 + xWidth, nil
	}

	return 0, fmt.Errorf("Could not get width of node %+v", node)
}
