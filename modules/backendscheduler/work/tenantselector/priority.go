package tenantselector

import (
	"container/heap"
	"sort"
)

type Item struct {
	value    string // tenantID
	priority int
	index    int // Index in the heap.
}

func NewItem(value string, priority int) *Item {
	return &Item{
		value:    value,
		priority: priority,
	}
}

func (i *Item) Value() string {
	return i.value
}

func (i *Item) Priority() int {
	return i.priority
}

var _ heap.Interface = (*PriorityQueue)(nil)

type PriorityQueue []*Item

func NewPriorityQueue() *PriorityQueue {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	return &pq
}

func (pq PriorityQueue) Items() []Item {
	items := make([]Item, len(pq))
	for i, item := range pq {
		items[i] = *item
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].priority > items[j].priority
	})

	return items
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) Update(item *Item, value string, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}

func (pq *PriorityQueue) UpdatePriority(selector TenantSelector) map[string]struct{} {
	values := make(map[string]struct{}, len(*pq))

	for _, item := range *pq {
		item.priority = selector.PriorityForTenant(item.value)
		pq.Update(item, item.value, item.priority)
		values[item.value] = struct{}{}
	}

	return values
}
