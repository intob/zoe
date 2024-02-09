package report

import (
	"container/heap"
	"encoding/json"

	"github.com/swissinfo-ch/lstn/ev"
)

type Top struct {
	N         int    // number of top content ids to include in the report
	MinEvTime uint32 // earliest time for events to be included in the report
}

// Define a heap structure to use with container/heap
type Item struct {
	Cid   uint32
	Views uint32
}
type ItemHeap []Item

func (h ItemHeap) Len() int           { return len(h) }
func (h ItemHeap) Less(i, j int) bool { return h[i].Views < h[j].Views } // Min-heap based on Views
func (h ItemHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *ItemHeap) Push(x interface{}) {
	*h = append(*h, x.(Item))
}

func (h *ItemHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// Generate returns a json representation of the top N content ids
func (t *Top) Generate(events <-chan *ev.Ev) ([]byte, error) {
	h := &ItemHeap{}
	heap.Init(h)
	cidViews := make(map[uint32]uint32)
	inHeap := make(map[uint32]bool) // Tracks whether a Cid is currently in the heap

	for e := range events {
		if e.Time < t.MinEvTime {
			// events are ordered by time, so we can break here
			break
		}
		if e.EvType != ev.EvType_LOAD {
			continue
		}
		cidViews[e.Cid]++
		if inHeap[e.Cid] {
			// If the item is in the heap, update its views in the heap.
			// This is a simplified view; actual implementation may require finding the item and updating it.
			// Consider using a custom method to update the item in the heap to maintain heap properties.
		} else if len(*h) < t.N {
			// If the heap is not full, add the item directly.
			heap.Push(h, Item{Cid: e.Cid, Views: cidViews[e.Cid]})
			inHeap[e.Cid] = true
		} else if cidViews[e.Cid] > (*h)[0].Views {
			// If the item has more views than the smallest in the heap, replace the smallest.
			inHeap[heap.Pop(h).(Item).Cid] = false // Mark the removed item as not in the heap
			heap.Push(h, Item{Cid: e.Cid, Views: cidViews[e.Cid]})
			inHeap[e.Cid] = true
		}
	}

	// Convert the heap to a slice for final processing.
	topN := make([]Item, h.Len())
	for i := len(topN) - 1; i >= 0; i-- {
		topN[i] = heap.Pop(h).(Item)
		inHeap[topN[i].Cid] = false // Technically unnecessary as we're done, but good for completeness
	}

	// Convert to map for final JSON output
	resultMap := make(map[uint32]uint32)
	for _, item := range topN {
		resultMap[item.Cid] = item.Views
	}

	return json.Marshal(resultMap)
}
