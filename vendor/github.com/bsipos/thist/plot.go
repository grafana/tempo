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
	"os"
	"strconv"
	"strings"
)

// Plot is a general plotting function for bar plots. It is used by Bar and BarSimple.
func Plot(x, y []float64, xlab, ylab []string, title string, info []string, symbol, negSymbol, space, top, vbar, hbar, tvbar string) string {
	if len(x) == 0 {
		return ""
	}
	// Based on: http://pyinsci.blogspot.com/2009/10/ascii-histograms.html
	width, height, terr := terminal.GetSize(int(os.Stderr.Fd()))
	if terr != nil {
		width = 80
		height = 24
	}

	xll := stringsMaxLen(xlab)
	yll := stringsMaxLen(ylab)
	width -= yll + 1

	res := strings.Repeat(space, yll+1) + centerPad2Len(title, space, int(width)) + "\n"
	height -= 4
	height -= len(info)

	height -= xll + 1

	xf := xFactor(len(x), int(width))
	if xf < 1 {
		xf = 1
	}

	if xll < xf-2 {
		height += xll - 1
	}

	ny := normalizeY(y, int(height))

	block := strings.Repeat(symbol, xf)
	nblock := strings.Repeat(negSymbol, xf)
	if xf > 2 {
		block = vbar + strings.Repeat(symbol, xf-2) + vbar
		nblock = vbar + strings.Repeat(negSymbol, xf-2) + vbar
	}

	blank := strings.Repeat(space, xf)
	topBar := strings.Repeat(top, xf)

	for l := int(height); l > 0; l-- {
		if yll > 0 {
			found := false
			for i, t := range ny {
				if l == t {
					res += fmt.Sprintf("%-"+strconv.Itoa(yll)+"s"+tvbar, ylab[i])
					found = true
					break
				}
			}
			if !found {
				res += strings.Repeat(space, yll) + vbar
			}
		}
		for _, c := range ny {
			if l == abs(c) {
				res += topBar
			} else if l < abs(c) {
				if c < 0 {
					res += nblock
				} else {
					res += block
				}
			} else {
				res += blank
			}
		}
		res += "\n"
	}

	if xll > 0 {
		res += strings.Repeat(space, yll) + vbar + strings.Repeat(hbar, int(width)) + "\n"
		if xll < xf-2 {
			res += strings.Repeat(space, yll) + vbar
			for _, xl := range xlab {
				res += vbar + rightPad2Len(xl, space, xf-1)
			}
		} else {
			for i := 0; i < xll; i++ {
				res += strings.Repeat(space, yll) + vbar
				for j := yll + 1; j < int(width); j++ {
					if (j-yll-1)%xf == 0 {
						bin := (j - yll - 1) / xf
						if bin < len(xlab) && i < len(xlab[bin]) {
							res += string(xlab[bin][i])
						} else {
							res += space
						}
					} else {
						res += space
					}
				}
				res += "\n"
			}

		}
	}
	for _, il := range info {
		res += strings.Repeat(space, yll) + vbar + centerPad2Len(il, space, int(width)) + "\n"
	}
	return res
}

// normalizeY normalizes y values to a maximum height.
func normalizeY(y []float64, height int) []int {
	max := max(y)
	res := make([]int, len(y))

	for i, x := range y {
		res[i] = int(x / max * float64(height))
	}
	return res
}

// xFactor.
func xFactor(n int, width int) int {
	return int(width / n)
}
