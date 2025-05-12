package renderer

import (
	"fmt"
	"github.com/olekukonko/ll"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter/tw"
)

// Blueprint implements a primary table rendering engine with customizable borders and alignments.
type Blueprint struct {
	config tw.Rendition // Rendering configuration for table borders and symbols
	logger *ll.Logger   // Logger for debug trace messages
}

// NewBlueprint creates a new Blueprint instance with optional custom configurations.
func NewBlueprint(configs ...tw.Rendition) *Blueprint {
	// Initialize with default configuration
	cfg := defaultBlueprint()
	if len(configs) > 0 {
		userCfg := configs[0]
		// Override default borders if provided
		if userCfg.Borders.Left != 0 {
			cfg.Borders.Left = userCfg.Borders.Left
		}
		if userCfg.Borders.Right != 0 {
			cfg.Borders.Right = userCfg.Borders.Right
		}
		if userCfg.Borders.Top != 0 {
			cfg.Borders.Top = userCfg.Borders.Top
		}
		if userCfg.Borders.Bottom != 0 {
			cfg.Borders.Bottom = userCfg.Borders.Bottom
		}
		// Override symbols if provided
		if userCfg.Symbols != nil {
			cfg.Symbols = userCfg.Symbols
		}

		// Merge user settings with default settings
		cfg.Settings = mergeSettings(cfg.Settings, userCfg.Settings)
	}
	return &Blueprint{config: cfg}
}

// Close performs cleanup (no-op in this implementation).
func (f *Blueprint) Close(w io.Writer) error {
	f.logger.Debug("Blueprint.Close() called (no-op).")
	return nil
}

// Config returns the renderer's current configuration.
func (f *Blueprint) Config() tw.Rendition {
	return f.config
}

// Footer renders the table footer section with configured formatting.
func (f *Blueprint) Footer(w io.Writer, footers [][]string, ctx tw.Formatting) {
	f.logger.Debug("Starting Footer render: IsSubRow=%v, Location=%v, Pos=%s",
		ctx.IsSubRow, ctx.Row.Location, ctx.Row.Position)
	// Render the footer line
	f.renderLine(w, ctx)
	f.logger.Debug("Completed Footer render")
}

// Header renders the table header section with configured formatting.
func (f *Blueprint) Header(w io.Writer, headers [][]string, ctx tw.Formatting) {
	f.logger.Debug("Starting Header render: IsSubRow=%v, Location=%v, Pos=%s, lines=%d, widths=%v",
		ctx.IsSubRow, ctx.Row.Location, ctx.Row.Position, len(ctx.Row.Current), ctx.Row.Widths)
	// Render the header line
	f.renderLine(w, ctx)
	f.logger.Debug("Completed Header render")
}

// Line renders a full horizontal row line with junctions and segments.
func (f *Blueprint) Line(w io.Writer, ctx tw.Formatting) {
	// Initialize junction renderer
	jr := NewJunction(JunctionContext{
		Symbols:       f.config.Symbols,
		Ctx:           ctx,
		ColIdx:        0,
		Logger:        f.logger,
		BorderTint:    Tint{},
		SeparatorTint: Tint{},
	})

	var line strings.Builder
	// Get sorted column indices
	sortedKeys := ctx.Row.Widths.SortedKeys()
	numCols := 0
	if len(sortedKeys) > 0 {
		numCols = sortedKeys[len(sortedKeys)-1] + 1
	}

	// Handle empty row case
	if numCols == 0 {
		prefix := ""
		suffix := ""
		if f.config.Borders.Left.Enabled() {
			prefix = jr.RenderLeft()
		}
		if f.config.Borders.Right.Enabled() {
			suffix = jr.RenderRight(-1)
		}
		if prefix != "" || suffix != "" {
			line.WriteString(prefix + suffix + tw.NewLine)
			fmt.Fprint(w, line.String())
		}
		f.logger.Debug("Line: Handled empty row/widths case")
		return
	}

	// Add left border if enabled
	if f.config.Borders.Left.Enabled() {
		line.WriteString(jr.RenderLeft())
	}

	totalWidth := 0
	visibleColIndices := make([]int, 0)
	// Calculate total width and visible columns
	for i, colIdx := range sortedKeys {
		colWidth := ctx.Row.Widths.Get(colIdx)
		if colWidth > 0 {
			visibleColIndices = append(visibleColIndices, colIdx)
			totalWidth += colWidth
			if i > 0 && f.config.Settings.Separators.BetweenColumns.Enabled() {
				prevColIdx := sortedKeys[i-1]
				prevColWidth := ctx.Row.Widths.Get(prevColIdx)
				if prevColWidth > 0 {
					totalWidth += tw.DisplayWidth(jr.sym.Column())
				}
			}
		}
	}

	f.logger.Debug("Line: sortedKeys=%v, Widths=%v, visibleColIndices=%v", sortedKeys, ctx.Row.Widths, visibleColIndices)
	// Render each column segment
	for keyIndex, currentColIdx := range visibleColIndices {
		jr.colIdx = currentColIdx
		segment := jr.GetSegment()
		colWidth := ctx.Row.Widths.Get(currentColIdx)
		f.logger.Debug("Line: colIdx=%d, segment='%s', width=%d", currentColIdx, segment, colWidth)
		if segment == "" {
			line.WriteString(strings.Repeat(" ", colWidth))
		} else {
			repeat := colWidth / tw.DisplayWidth(segment)
			if repeat < 1 && colWidth > 0 {
				repeat = 1
			}
			line.WriteString(strings.Repeat(segment, repeat))
		}

		// Add junction between columns if not the last column
		isLast := keyIndex == len(visibleColIndices)-1
		if !isLast && f.config.Settings.Separators.BetweenColumns.Enabled() {
			nextColIdx := visibleColIndices[keyIndex+1]
			junction := jr.RenderJunction(currentColIdx, nextColIdx)
			f.logger.Debug("Line: Junction between %d and %d: '%s'", currentColIdx, nextColIdx, junction)
			line.WriteString(junction)
		}
	}

	// Add right border and adjust width if necessary
	if f.config.Borders.Right.Enabled() && len(visibleColIndices) > 0 {
		lastIdx := visibleColIndices[len(visibleColIndices)-1]
		line.WriteString(jr.RenderRight(lastIdx))
		actualWidth := tw.DisplayWidth(line.String()) - tw.DisplayWidth(jr.RenderLeft()) - tw.DisplayWidth(jr.RenderRight(lastIdx))
		if actualWidth > totalWidth {
			lineStr := line.String()
			line.Reset()
			excess := actualWidth - totalWidth
			line.WriteString(lineStr[:len(lineStr)-excess-tw.DisplayWidth(jr.RenderRight(lastIdx))] + jr.RenderRight(lastIdx))
		}
	}

	// Write the final line
	line.WriteString(tw.NewLine)
	fmt.Fprint(w, line.String())
	f.logger.Debug("Line rendered: %s", strings.TrimSuffix(line.String(), tw.NewLine))
}

// Logger sets the logger for the Blueprint instance.
func (f *Blueprint) Logger(logger *ll.Logger) {
	f.logger = logger.Namespace("blueprint")
}

// Row renders a table data row with configured formatting.
func (f *Blueprint) Row(w io.Writer, row []string, ctx tw.Formatting) {
	f.logger.Debug("Starting Row render: IsSubRow=%v, Location=%v, Pos=%s, hasFooter=%v",
		ctx.IsSubRow, ctx.Row.Location, ctx.Row.Position, ctx.HasFooter)
	// Render the row line
	f.renderLine(w, ctx)
	f.logger.Debug("Completed Row render")
}

// Start initializes the rendering process (no-op in this implementation).
func (f *Blueprint) Start(w io.Writer) error {
	f.logger.Debug("Blueprint.Start() called (no-op).")
	return nil
}

// formatCell formats a cell's content with specified width, padding, and alignment, returning an empty string if width is non-positive.
func (f *Blueprint) formatCell(content string, width int, padding tw.Padding, align tw.Align) string {
	if width <= 0 {
		return ""
	}

	f.logger.Debug("Formatting cell: content='%s', width=%d, align=%s, padding={L:'%s' R:'%s'}",
		content, width, align, padding.Left, padding.Right)

	// Calculate display width of content
	runeWidth := tw.DisplayWidth(content)

	// Set default padding characters
	leftPadChar := padding.Left
	rightPadChar := padding.Right
	if leftPadChar == "" {
		leftPadChar = " "
	}
	if rightPadChar == "" {
		rightPadChar = " "
	}

	// Calculate padding widths
	padLeftWidth := tw.DisplayWidth(leftPadChar)
	padRightWidth := tw.DisplayWidth(rightPadChar)

	// Calculate available width for content
	availableContentWidth := width - padLeftWidth - padRightWidth
	if availableContentWidth < 0 {
		availableContentWidth = 0
	}
	f.logger.Debug("Available content width: %d", availableContentWidth)

	// Truncate content if it exceeds available width
	if runeWidth > availableContentWidth {
		content = tw.TruncateString(content, availableContentWidth)
		runeWidth = tw.DisplayWidth(content)
		f.logger.Debug("Truncated content to fit %d: '%s' (new width %d)", availableContentWidth, content, runeWidth)
	}

	// Calculate total padding needed
	totalPaddingWidth := width - runeWidth
	if totalPaddingWidth < 0 {
		totalPaddingWidth = 0
	}
	f.logger.Debug("Total padding width: %d", totalPaddingWidth)

	var result strings.Builder
	var leftPaddingWidth, rightPaddingWidth int

	// Apply alignment and padding
	switch align {
	case tw.AlignLeft:
		result.WriteString(leftPadChar)
		result.WriteString(content)
		rightPaddingWidth = totalPaddingWidth - padLeftWidth
		if rightPaddingWidth > 0 {
			result.WriteString(strings.Repeat(rightPadChar, rightPaddingWidth/padRightWidth))
			leftover := rightPaddingWidth % padRightWidth
			if leftover > 0 {
				result.WriteString(strings.Repeat(" ", leftover))
				f.logger.Debug("Added %d leftover spaces for right padding", leftover)
			}
			f.logger.Debug("Applied right padding: '%s' for %d width", rightPadChar, rightPaddingWidth)
		}
	case tw.AlignRight:
		leftPaddingWidth = totalPaddingWidth - padRightWidth
		if leftPaddingWidth > 0 {
			result.WriteString(strings.Repeat(leftPadChar, leftPaddingWidth/padLeftWidth))
			leftover := leftPaddingWidth % padLeftWidth
			if leftover > 0 {
				result.WriteString(strings.Repeat(" ", leftover))
				f.logger.Debug("Added %d leftover spaces for left padding", leftover)
			}
			f.logger.Debug("Applied left padding: '%s' for %d width", leftPadChar, leftPaddingWidth)
		}
		result.WriteString(content)
		result.WriteString(rightPadChar)
	case tw.AlignCenter:
		leftPaddingWidth = (totalPaddingWidth-padLeftWidth-padRightWidth)/2 + padLeftWidth
		rightPaddingWidth = totalPaddingWidth - leftPaddingWidth
		if leftPaddingWidth > padLeftWidth {
			result.WriteString(strings.Repeat(leftPadChar, (leftPaddingWidth-padLeftWidth)/padLeftWidth))
			leftover := (leftPaddingWidth - padLeftWidth) % padLeftWidth
			if leftover > 0 {
				result.WriteString(strings.Repeat(" ", leftover))
				f.logger.Debug("Added %d leftover spaces for left centering", leftover)
			}
			f.logger.Debug("Applied left centering padding: '%s' for %d width", leftPadChar, leftPaddingWidth-padLeftWidth)
		}
		result.WriteString(leftPadChar)
		result.WriteString(content)
		result.WriteString(rightPadChar)
		if rightPaddingWidth > padRightWidth {
			result.WriteString(strings.Repeat(rightPadChar, (rightPaddingWidth-padRightWidth)/padRightWidth))
			leftover := (rightPaddingWidth - padRightWidth) % padRightWidth
			if leftover > 0 {
				result.WriteString(strings.Repeat(" ", leftover))
				f.logger.Debug("Added %d leftover spaces for right centering", leftover)
			}
			f.logger.Debug("Applied right centering padding: '%s' for %d width", rightPadChar, rightPaddingWidth-padRightWidth)
		}
	default:
		// Default to left alignment
		result.WriteString(leftPadChar)
		result.WriteString(content)
		rightPaddingWidth = totalPaddingWidth - padLeftWidth
		if rightPaddingWidth > 0 {
			result.WriteString(strings.Repeat(rightPadChar, rightPaddingWidth/padRightWidth))
			leftover := rightPaddingWidth % padRightWidth
			if leftover > 0 {
				result.WriteString(strings.Repeat(" ", leftover))
				f.logger.Debug("Added %d leftover spaces for right padding", leftover)
			}
			f.logger.Debug("Applied right padding: '%s' for %d width", rightPadChar, rightPaddingWidth)
		}
	}

	output := result.String()
	finalWidth := tw.DisplayWidth(output)
	// Adjust output to match target width
	if finalWidth > width {
		output = tw.TruncateString(output, width)
		f.logger.Debug("formatCell: Truncated output to width %d", width)
	} else if finalWidth < width {
		result.WriteString(strings.Repeat(" ", width-finalWidth))
		output = result.String()
		f.logger.Debug("formatCell: Added %d spaces to meet width %d", width-finalWidth, width)
	}

	// Log warning if final width doesn't match target
	if f.logger.Enabled() && tw.DisplayWidth(output) != width {
		f.logger.Debug("formatCell Warning: Final width %d does not match target %d for result '%s'",
			tw.DisplayWidth(output), width, output)
	}

	f.logger.Debug("Formatted cell final result: '%s' (target width %d)", output, width)
	return output
}

// renderLine renders a single line (header, row, or footer) with borders, separators, and merge handling.
func (f *Blueprint) renderLine(w io.Writer, ctx tw.Formatting) {
	// Get sorted column indices
	sortedKeys := ctx.Row.Widths.SortedKeys()
	numCols := 0
	if len(sortedKeys) > 0 {
		numCols = sortedKeys[len(sortedKeys)-1] + 1
	} else {
		// Handle empty row case
		prefix := ""
		if f.config.Borders.Left.Enabled() {
			prefix = f.config.Symbols.Column()
		}
		suffix := ""
		if f.config.Borders.Right.Enabled() {
			suffix = f.config.Symbols.Column()
		}
		if prefix != "" || suffix != "" {
			fmt.Fprintln(w, prefix+suffix)
		}
		f.logger.Debug("renderLine: Handled empty row/widths case.")
		return
	}

	// Set column separator and borders
	columnSeparator := f.config.Symbols.Column()
	prefix := ""
	if f.config.Borders.Left.Enabled() {
		prefix = columnSeparator
	}
	suffix := ""
	if f.config.Borders.Right.Enabled() {
		suffix = columnSeparator
	}

	var output strings.Builder
	output.WriteString(prefix)

	colIndex := 0
	separatorDisplayWidth := 0
	if f.config.Settings.Separators.BetweenColumns.Enabled() {
		separatorDisplayWidth = tw.DisplayWidth(columnSeparator)
	}

	// Process each column
	for colIndex < numCols {
		visualWidth := ctx.Row.Widths.Get(colIndex)
		cellCtx, ok := ctx.Row.Current[colIndex]
		isHMergeStart := ok && cellCtx.Merge.Horizontal.Present && cellCtx.Merge.Horizontal.Start
		if visualWidth == 0 && !isHMergeStart {
			f.logger.Debug("renderLine: Skipping col %d (zero width, not HMerge start)", colIndex)
			colIndex++
			continue
		}

		// Determine if a separator is needed
		shouldAddSeparator := false
		if colIndex > 0 && f.config.Settings.Separators.BetweenColumns.Enabled() {
			prevWidth := ctx.Row.Widths.Get(colIndex - 1)
			prevCellCtx, prevOk := ctx.Row.Current[colIndex-1]
			prevIsHMergeEnd := prevOk && prevCellCtx.Merge.Horizontal.Present && prevCellCtx.Merge.Horizontal.End
			if (prevWidth > 0 || prevIsHMergeEnd) && (!ok || !(cellCtx.Merge.Horizontal.Present && !cellCtx.Merge.Horizontal.Start)) {
				shouldAddSeparator = true
			}
		}
		if shouldAddSeparator {
			output.WriteString(columnSeparator)
			f.logger.Debug("renderLine: Added separator '%s' before col %d", columnSeparator, colIndex)
		} else if colIndex > 0 {
			f.logger.Debug("renderLine: Skipped separator before col %d due to zero-width prev col or HMerge continuation", colIndex)
		}

		// Handle merged cells
		span := 1
		if isHMergeStart {
			span = cellCtx.Merge.Horizontal.Span
			if ctx.Row.Position == tw.Row {
				dynamicTotalWidth := 0
				for k := 0; k < span && colIndex+k < numCols; k++ {
					normWidth := ctx.NormalizedWidths.Get(colIndex + k)
					if normWidth < 0 {
						normWidth = 0
					}
					dynamicTotalWidth += normWidth
					if k > 0 && separatorDisplayWidth > 0 && ctx.NormalizedWidths.Get(colIndex+k) > 0 {
						dynamicTotalWidth += separatorDisplayWidth
					}
				}
				visualWidth = dynamicTotalWidth
				f.logger.Debug("renderLine: Row HMerge col %d, span %d, dynamic visualWidth %d", colIndex, span, visualWidth)
			} else {
				visualWidth = ctx.Row.Widths.Get(colIndex)
				f.logger.Debug("renderLine: H/F HMerge col %d, span %d, pre-adjusted visualWidth %d", colIndex, span, visualWidth)
			}
		} else {
			visualWidth = ctx.Row.Widths.Get(colIndex)
			f.logger.Debug("renderLine: Regular col %d, visualWidth %d", colIndex, visualWidth)
		}
		if visualWidth < 0 {
			visualWidth = 0
		}

		// Skip processing for non-start merged cells
		if ok && cellCtx.Merge.Horizontal.Present && !cellCtx.Merge.Horizontal.Start {
			f.logger.Debug("renderLine: Skipping col %d processing (part of HMerge)", colIndex)
			colIndex++
			continue
		}

		// Handle empty cell context
		if !ok {
			if visualWidth > 0 {
				output.WriteString(strings.Repeat(" ", visualWidth))
				f.logger.Debug("renderLine: No cell context for col %d, writing %d spaces", colIndex, visualWidth)
			} else {
				f.logger.Debug("renderLine: No cell context for col %d, visualWidth is 0, writing nothing", colIndex)
			}
			colIndex += span
			continue
		}

		// Set cell padding and alignment
		padding := cellCtx.Padding
		align := cellCtx.Align
		if align == tw.AlignNone {
			if ctx.Row.Position == tw.Header {
				align = tw.AlignCenter
			} else if ctx.Row.Position == tw.Footer {
				align = tw.AlignRight
			} else {
				align = tw.AlignLeft
			}
			f.logger.Debug("renderLine: col %d (data: '%s') using renderer default align '%s' for position %s.", colIndex, cellCtx.Data, align, ctx.Row.Position)
		} else if align == tw.Skip {
			if ctx.Row.Position == tw.Header {
				align = tw.AlignCenter
			} else if ctx.Row.Position == tw.Footer {
				align = tw.AlignRight
			} else {
				align = tw.AlignLeft
			}
			f.logger.Debug("renderLine: col %d (data: '%s') cellCtx.Align was Skip/empty, falling back to basic default '%s'.", colIndex, cellCtx.Data, align)
		}

		isTotalPattern := false

		// Override alignment for footer merged cells
		if (ctx.Row.Position == tw.Footer && isHMergeStart) || isTotalPattern {
			if align != tw.AlignRight {
				f.logger.Debug("renderLine: Applying AlignRight HMerge/TOTAL override for Footer col %d. Original/default align was: %s", colIndex, align)
				align = tw.AlignRight
			}
		}

		// Handle vertical/hierarchical merges
		cellData := cellCtx.Data
		if (cellCtx.Merge.Vertical.Present && !cellCtx.Merge.Vertical.Start) ||
			(cellCtx.Merge.Hierarchical.Present && !cellCtx.Merge.Hierarchical.Start) {
			cellData = ""
			f.logger.Debug("renderLine: Blanked data for col %d (non-start V/Hierarchical)", colIndex)
		}

		// Format and render the cell
		formattedCell := f.formatCell(cellData, visualWidth, padding, align)
		if len(formattedCell) > 0 {
			output.WriteString(formattedCell)
		}

		// Log rendering details
		if isHMergeStart {
			f.logger.Debug("renderLine: Rendered HMerge START col %d (span %d, visualWidth %d, align %v): '%s'",
				colIndex, span, visualWidth, align, formattedCell)
		} else {
			f.logger.Debug("renderLine: Rendered regular col %d (visualWidth %d, align %v): '%s'",
				colIndex, visualWidth, align, formattedCell)
		}
		colIndex += span
	}

	// Add suffix and write the line
	if output.Len() > len(prefix) || f.config.Borders.Right.Enabled() {
		output.WriteString(suffix)
	}
	output.WriteString(tw.NewLine)
	fmt.Fprint(w, output.String())
	f.logger.Debug("renderLine: Final rendered line: %s", strings.TrimSuffix(output.String(), tw.NewLine))
}
