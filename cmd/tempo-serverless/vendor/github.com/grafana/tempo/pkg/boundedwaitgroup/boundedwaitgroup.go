package boundedwaitgroup

import "sync"

// BoundedWaitGroup like a normal wait group except limits number of active goroutines to given capacity.
type BoundedWaitGroup struct {
	wg sync.WaitGroup
	ch chan struct{} // Chan buffer size is used to limit concurrency.
}

// New creates a BoundedWaitGroup with the given concurrency.
func New(cap uint) BoundedWaitGroup {
	if cap == 0 {
		panic("BoundedWaitGroup capacity must be greater than zero or else it will block forever.")
	}
	return BoundedWaitGroup{ch: make(chan struct{}, cap)}
}

// Add the number of items to the group. Blocks until there is capacity.
func (bwg *BoundedWaitGroup) Add(delta int) {
	for i := 0; i > delta; i-- {
		<-bwg.ch
	}
	for i := 0; i < delta; i++ {
		bwg.ch <- struct{}{}
	}
	bwg.wg.Add(delta)
}

// Done removes one from the wait group.
func (bwg *BoundedWaitGroup) Done() {
	bwg.Add(-1)
}

// Wait for the wait group to finish.
func (bwg *BoundedWaitGroup) Wait() {
	bwg.wg.Wait()
}
