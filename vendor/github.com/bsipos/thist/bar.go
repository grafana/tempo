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

// BarSimple plots a bar plot on the terminal. Does not use unicode charcters.
func BarSimple(x, y []float64, xlab, ylab []string, title string, info []string) string {
	if len(xlab) == 0 {
		xlab = AutoLabel(x, mean(absFloats(x)))
	}
	if len(ylab) == 0 {
		ylab = AutoLabel(y, mean(absFloats(y)))
	}
	return Plot(x, y, xlab, ylab, title, info, "#", "@", " ", "_", "|", "-", "|")
}

// Bar plots a bar plot on the terminal. It makes use of unicode characters.
func Bar(x, y []float64, xlab, ylab []string, title string, info []string) string {
	if len(xlab) == 0 {
		xlab = AutoLabel(x, mean(absFloats(x)))
	}
	if len(ylab) == 0 {
		ylab = AutoLabel(y, mean(absFloats(y)))
	}
	return Plot(x, y, xlab, ylab, title, info, "\u2588", "\u2591", " ", "_", "\u2502", "\u2500", "\u2524")
}
