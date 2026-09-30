// Package render turns a comparison into views of it, for each output to show:
// a summary of every case, a case's box plots and table, and how the runs
// differ.
//
// A view holds no layout, and says what each piece means rather than how it
// looks, so every output shows it the same way by its own means. Package term
// lays views out for a terminal, and package markdown for a pull request.
package render
