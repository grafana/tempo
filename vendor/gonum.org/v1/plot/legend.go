// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"math"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// A Legend gives a description of the meaning of different
// data elements of the plot.  Each legend entry has a name
// and a thumbnail, where the thumbnail shows a small
// sample of the display style of the corresponding data.
type Legend struct {
	// TextStyle is the style given to the legend
	// entry texts.
	draw.TextStyle

	// Padding is the amount of padding to add
	// between each entry in the legend.  If Padding
	// is zero then entries are spaced based on the
	// font size.
	Padding vg.Length

	// Top and Left specify the location of the legend.
	// If Top is true the legend is located along the top
	// edge of the plot, otherwise it is located along
	// the bottom edge.  If Left is true then the legend
	// is located along the left edge of the plot, and the
	// text is positioned after the icons, otherwise it is
	// located along the right edge and the text is
	// positioned before the icons.
	Top, Left bool

	// XOffs and YOffs are added to the legend's
	// final position.
	XOffs, YOffs vg.Length

	// ThumbnailWidth is the width of legend thumbnails.
	ThumbnailWidth vg.Length

	// entries are all of the legendEntries described
	// by this legend.
	entries []legendEntry
}

// A legendEntry represents a single line of a legend, it
// has a name and an icon.
type legendEntry struct {
	// text is the text associated with this entry.
	text string

	// thumbs is a slice of all of the thumbnails styles
	thumbs []Thumbnailer
}

// Thumbnailer wraps the Thumbnail method, which
// draws the small image in a legend representing the
// style of data.
type Thumbnailer interface {
	// Thumbnail draws an thumbnail representing
	// a legend entry.  The thumbnail will usually show
	// a smaller representation of the style used
	// to plot the corresponding data.
	Thumbnail(c *draw.Canvas)
}

// NewLegend returns a legend with the default
// parameter settings.
func NewLegend() (Legend, error) {
	font, err := vg.MakeFont(DefaultFont, vg.Points(12))
	if err != nil {
		return Legend{}, err
	}
	return Legend{
		ThumbnailWidth: vg.Points(20),
		TextStyle:      draw.TextStyle{Font: font},
	}, nil
}

// Draw draws the legend to the given draw.Canvas.
func (l *Legend) Draw(c draw.Canvas) {
	iconx := c.Min.X
	sty := l.TextStyle
	textx := iconx + l.ThumbnailWidth + sty.Rectangle(" ").Max.X
	if !l.Left {
		iconx = c.Max.X - l.ThumbnailWidth
		textx = iconx - l.TextStyle.Rectangle(" ").Max.X
		sty.XAlign--
	}
	textx += l.XOffs
	iconx += l.XOffs

	enth := l.entryHeight()
	y := c.Max.Y - enth
	if !l.Top {
		y = c.Min.Y + (enth+l.Padding)*(vg.Length(len(l.entries))-1)
	}
	y += l.YOffs

	icon := &draw.Canvas{
		Canvas: c.Canvas,
		Rectangle: vg.Rectangle{
			Min: vg.Point{X: iconx, Y: y},
			Max: vg.Point{X: iconx + l.ThumbnailWidth, Y: y + enth},
		},
	}
	for _, e := range l.entries {
		for _, t := range e.thumbs {
			t.Thumbnail(icon)
		}
		yoffs := (enth - sty.Rectangle(e.text).Max.Y) / 2
		c.FillText(sty, vg.Point{X: textx, Y: icon.Min.Y + yoffs}, e.text)
		icon.Min.Y -= enth + l.Padding
		icon.Max.Y -= enth + l.Padding
	}
}

// Rectangle returns the extent of the Legend.
func (l *Legend) Rectangle(c draw.Canvas) vg.Rectangle {
	var width, height vg.Length
	sty := l.TextStyle
	entryHeight := l.entryHeight()
	for i, e := range l.entries {
		width = vg.Length(math.Max(float64(width), float64(l.ThumbnailWidth+sty.Rectangle(" "+e.text).Max.X)))
		height += entryHeight
		if i != 0 {
			height += l.Padding
		}
	}
	var r vg.Rectangle
	if l.Left {
		r.Max.X = c.Max.X
		r.Min.X = c.Max.X - width
	} else {
		r.Max.X = c.Min.X + width
		r.Min.X = c.Min.X
	}
	if l.Top {
		r.Max.Y = c.Max.Y
		r.Min.Y = c.Max.Y - height
	} else {
		r.Max.Y = c.Min.Y + height
		r.Min.Y = c.Min.Y
	}
	return r
}

// entryHeight returns the height of the tallest legend
// entry text.
func (l *Legend) entryHeight() (height vg.Length) {
	for _, e := range l.entries {
		if h := l.TextStyle.Rectangle(e.text).Max.Y; h > height {
			height = h
		}
	}
	return
}

// Add adds an entry to the legend with the given name.
// The entry's thumbnail is drawn as the composite of all of the
// thumbnails.
func (l *Legend) Add(name string, thumbs ...Thumbnailer) {
	l.entries = append(l.entries, legendEntry{text: name, thumbs: thumbs})
}
