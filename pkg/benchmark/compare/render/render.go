// Package render turns a comparison into something to read: tables and box
// plots for a terminal, and markdown for a pull request.
//
// It holds no terminal state. Renderers return plain strings, or lines of
// segments that say what each piece shows, so an interactive view can style
// them and anything else can print them as they are.
package render
