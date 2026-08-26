package store

type Side string

const (
	Client Side = "client"
	Server Side = "server"
)

// Callback is invoked by the store with a pooled *Edge while holding the
// store mutex. The Edge pointer is only valid for the duration of the call —
// the store may return it to the pool after the callback returns. Do not
// retain the pointer, send it to another goroutine, or store substrings of
// its fields beyond the callback's lifetime.
type Callback func(e *Edge)
