// Package render turns a comparison into views of it, for each output to show:
// a summary of every case, and how the runs differ.
//
// A view holds no layout, and says what each piece means rather than how it
// looks, so every output shows it the same way by its own means. Package
// markdown writes views for a pull request.
package render
