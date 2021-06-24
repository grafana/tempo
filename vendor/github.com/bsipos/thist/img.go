// Copyright © 2019 Botond Sipos
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package thist

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// SaveImage saves a histogram to an image file using gonum plot.
func (h *Hist) SaveImage(f string) {
	data := plotter.Values(h.Counts)

	if h.Normalize {
		data = h.NormCounts()
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = h.Title
	p.Y.Label.Text = "Count"
	if h.Normalize {
		p.Y.Label.Text = "Frequency"
	}

	bins := make([]plotter.HistogramBin, len(h.BinStart))
	for i, binStart := range h.BinStart {
		bins[i] = plotter.HistogramBin{binStart, h.BinEnd[i], data[i]}
	}

	ph := &plotter.Histogram{
		Bins:      bins,
		Width:     h.DataMax - h.DataMin,
		FillColor: plotutil.Color(2),
		LineStyle: plotter.DefaultLineStyle,
	}
	ph.LineStyle.Width = vg.Length(0.5)
	ph.Color = plotutil.Color(0)

	p.Add(ph)
	p.X.Label.Text = h.Info

	if err := p.Save(11.69*vg.Inch, 8.27*vg.Inch, f); err != nil {
		panic(err)
	}
}
