package util

// Stack is a generic stack implementation that supports len() and cap().
type Stack[T any] []T

// Push pushes a new element on the stack.
func (s *Stack[T]) Push(element T) {
	*s = append(*s, element)
}

// Peek returns the top element from the stack without removing it. If the stack is
// empty the second return value is false.
func (s *Stack[T]) Peek() (T, bool) {
	if len(*s) == 0 {
		var zero T
		return zero, false
	}
	return (*s)[len(*s)-1], true
}

// Pop returns the top element from the stack and removes it. If the stack is empty
// the second return value is false.
func (s *Stack[T]) Pop() (T, bool) {
	if len(*s) == 0 {
		var zero T
		return zero, false
	}
	i := len(*s) - 1

	element := (*s)[i]
	*s = (*s)[:i]

	return element, true
}

// IsEmpty returns whether the stack contains elements
func (s *Stack[T]) IsEmpty() bool {
	return len(*s) == 0
}

// Reset empties the stack while maintaining its capacity
func (s *Stack[T]) Reset() {
	*s = (*s)[:0]
}
