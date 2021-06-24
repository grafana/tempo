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
	"fmt"
	terminal "golang.org/x/crypto/ssh/terminal"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Hist is a struct holding the parameters and internal state of a histogram object.
type Hist struct {
	Title        string
	BinMode      string
	MaxBins      int
	NrBins       int
	DataCount    int
	DataMap      map[float64]float64
	DataMin      float64
	DataMax      float64
	DataMean     float64
	DataSd       float64
	Normalize    bool
	BinStart     []float64
	BinEnd       []float64
	Counts       []float64
	m            float64
	MaxPrecision float64
	Precision    float64
	BinWidth     float64
	Info         string
}

// NewHist initilizes a new histogram object.  If data is not nil the data points are processed and the state is updated.
func NewHist(data []float64, title, binMode string, maxBins int, normalize bool) *Hist {
	h := &Hist{title, binMode, maxBins, 0, 0, make(map[float64]float64), math.NaN(), math.NaN(), math.NaN(), math.NaN(), normalize, []float64{}, []float64{}, []float64{}, math.NaN(), 14.0, 14.0, math.NaN(), ""}
	if h.BinMode == "" {
		h.BinMode = "termfit"
	}

	if len(data) > 0 {
		min, max := data[0], data[0]
		h.DataMean = data[0]
		h.DataSd = 0.0
		h.m = 0.0
		for _, d := range data {
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
			h.DataCount++
			h.updateMoments(d)
		}
		h.DataMin = min
		h.DataMax = max
		h.BinStart, h.BinEnd, h.BinWidth = h.buildBins()
		h.updatePrecision()
		h.Counts = make([]float64, len(h.BinStart))
		for _, v := range data {
			c := roundFloat64(v, h.Precision)
			h.DataMap[c]++
			i := sort.SearchFloat64s(h.BinStart, c) - 1
			if i < 0 {
				i = 0
			}
			h.Counts[i]++
		}
		h.updateInfo()
	}

	return h
}

// updateInfo updates the info string based on the current internal state.
func (h *Hist) updateInfo() {
	digits := strconv.Itoa(int(h.Precision))
	h.Info = fmt.Sprintf("Count: %d Mean: %."+digits+"f Stdev: %."+digits+"f Min: %."+digits+"f Max: %."+digits+"f Precision: %.0f Bins: %d\n", h.DataCount, h.DataMean, h.DataSd, h.DataMin, h.DataMax, h.Precision, len(h.BinStart))
}

func (h *Hist) buildBins() ([]float64, []float64, float64) {
	var n int
	var w float64

	if h.DataMin == h.DataMax {
		n = 1
		w = 1
	} else if h.BinMode == "fixed" {
		n = h.MaxBins
		w = (h.DataMax - h.DataMin) / float64(n)
	} else if h.BinMode == "auto" || h.BinMode == "fit" || h.BinMode == "termfit" {
		w = scottsRule(h.DataCount, h.DataSd)
		n = int((h.DataMax - h.DataMin) / w)
		if n < 1 {
			n = 1
		}
		if h.BinMode == "fit" && n > h.MaxBins {
			n = h.MaxBins
		}
		if h.BinMode == "termfit" {
			tm, _, terr := terminal.GetSize(int(os.Stderr.Fd()))
			if terr != nil {
				tm = 80
			}
			tm -= 10
			if n > int(tm) {
				n = int(tm)
			}

		}
		w = (h.DataMax - h.DataMin) / float64(n)
	}

	s := make([]float64, n)
	e := make([]float64, n)

	for i := 0; i < n; i++ {
		s[i] = h.DataMin + float64(i)*w
		e[i] = h.DataMin + float64(i+1)*w
	}

	return s, e, w

}

// NormCounts returns the normalised counts for each bin.
func (h *Hist) NormCounts() []float64 {
	res := make([]float64, len(h.Counts))
	for i, c := range h.Counts {
		res[i] = c / float64(h.DataCount) / h.BinWidth
	}
	return res
}

// updateMoments calculates the new mean and sd of the dataset after adding a new data point p.
func (h *Hist) updateMoments(p float64) {
	oldMean := h.DataMean
	h.DataMean += (p - h.DataMean) / float64(h.DataCount)
	h.m += (p - oldMean) * (p - h.DataMean)
	h.DataSd = math.Sqrt(h.m / float64(h.DataCount))
}

// scottsRule calculates the number of histogram bins based on Scott's rule:
// https://en.wikipedia.org/wiki/Histogram#Scott's_normal_reference_rule
func scottsRule(n int, sd float64) float64 {
	h := (3.5 * sd) / math.Pow(float64(n), 1.0/3.0)
	return h
}

// Update adds a new data point and updates internal state.
func (h *Hist) Update(p float64) {
	h.DataCount++
	oldMin := h.DataMin
	oldMax := h.DataMax
	if math.IsNaN(h.DataMin) || p < h.DataMin {
		h.DataMin = p
	}
	if math.IsNaN(h.DataMax) || p > h.DataMax {
		h.DataMax = p
	}
	if h.DataCount == 1 {
		h.DataMean = p
		h.DataSd = 0.0
		h.m = 0.0
		h.BinStart, h.BinEnd, h.BinWidth = h.buildBins()
		h.updatePrecision()
		h.Counts = []float64{1.0}
	} else {
		h.updateMoments(p)
		h.updateInfo()
	}

	h.DataMap[roundFloat64(p, h.Precision)]++

	if !math.IsNaN(oldMin) && p >= oldMin && !math.IsNaN(oldMax) && p <= oldMax {
		var i int
		if p == oldMin {
			i = 0
		} else if p == oldMax {
			i = len(h.Counts) - 1
		} else {
			i = sort.SearchFloat64s(h.BinStart, p) - 1
			if i < 0 {
				i = 0
			}
		}
		h.Counts[i]++
		return
	}

	h.BinStart, h.BinEnd, h.BinWidth = h.buildBins()
	h.updatePrecision()
	newCounts := make([]float64, len(h.BinStart))

	for v, c := range h.DataMap {
		i := sort.SearchFloat64s(h.BinStart, v) - 1
		if i < 0 {
			i = 0
		}
		newCounts[i] += c
	}

	h.Counts = newCounts

}

// updatePrecision claculates the precision to use for binnig based on the
// bin width and the maximum allowed precision.
func (h *Hist) updatePrecision() {
	h.Precision = math.Ceil(-math.Log10(h.BinWidth)) * 2.0
	if h.Precision > h.MaxPrecision {
		h.Precision = h.MaxPrecision
	}
	if h.Precision < 1.0 {
		h.Precision = 1.0
	}
}

// Draw calls Bar to draw the hsitogram to the terminal.
func (h *Hist) Draw() string {
	d := h.Counts
	if h.Normalize {
		d = h.NormCounts()
	}
	return Bar(h.BinStart, d, []string{}, []string{}, h.Title, strings.Split(strings.TrimRight(h.Info, "\n"), "\n"))
}

// DrawSimple calls BarSimple to draw the hsitogram to the terminal.
func (h *Hist) DrawSimple() string {
	d := h.Counts
	if h.Normalize {
		d = h.NormCounts()
	}
	return BarSimple(h.BinStart, d, []string{}, []string{}, h.Title, strings.Split(strings.TrimRight(h.Info, "\n"), "\n"))
}

// Summary return a string summary of the internal state of a Hist object.
func (h *Hist) Summary() string {
	res := "" // FIXME: TODO
	return res
}

// Dump prints the bins and counts to the standard output.
func (h *Hist) Dump() string {
	res := "Bin\tBinStart\tBinEnd\tCount\n"

	for i, c := range h.Counts {
		res += fmt.Sprintf("%d\t%.4f\t%.4f\t%.0f\n", i, h.BinStart[i], h.BinEnd[i], c)
	}

	return res
}
