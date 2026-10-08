// Package rows is the bookkeeping for drawing a long list a window at a
// time, as rush draws a transcript (convo/window.go): Index counts each
// item's rows so an item's first row, and the item at a row, are found by
// halving however long the list; Cache keeps drawings up to a budget of
// rows, letting the least recently drawn go first.
package rows

import "container/list"

// Index counts rows per item in a Fenwick tree. The zero value is empty.
type Index struct {
	h    []int // each item's rows
	tree []int // 1-based Fenwick tree over h
}

// Reset counts heights, one an item, in place of what was counted.
func (ix *Index) Reset(heights []int) {
	n := len(heights)
	ix.h = append(ix.h[:0], heights...)
	ix.tree = append(ix.tree[:0], make([]int, n+1)...)
	for i := 1; i <= n; i++ {
		ix.tree[i] += ix.h[i-1]
		if j := i + i&-i; j <= n {
			ix.tree[j] += ix.tree[i]
		}
	}
}

// Len is how many items are counted.
func (ix *Index) Len() int { return len(ix.h) }

// Height is item i's rows.
func (ix *Index) Height(i int) int { return ix.h[i] }

// Set counts item i as rows high.
func (ix *Index) Set(i, rows int) {
	d := rows - ix.h[i]
	if d == 0 {
		return
	}
	ix.h[i] = rows
	for i++; i < len(ix.tree); i += i & -i {
		ix.tree[i] += d
	}
}

// Prefix is the rows of items [0, i): item i's first row.
func (ix *Index) Prefix(i int) int {
	sum := 0
	for i = min(i, len(ix.h)); i > 0; i -= i & -i {
		sum += ix.tree[i]
	}
	return sum
}

// Total is every item's rows.
func (ix *Index) Total() int { return ix.Prefix(len(ix.h)) }

// Find is the item holding row r, the nearest one off either end.
func (ix *Index) Find(r int) int {
	n := len(ix.h)
	if n == 0 || r < 0 {
		return 0
	}
	i, step := 0, 1
	for step*2 <= n {
		step *= 2
	}
	for ; step > 0; step /= 2 {
		if j := i + step; j <= n && ix.tree[j] <= r {
			i, r = j, r-ix.tree[j]
		}
	}
	return min(i, n-1)
}

// Cache keeps drawings, each costing its rows, most recently used first.
type Cache[K comparable, V any] struct {
	budget int
	used   *list.List // of *entry[K, V], front the most recent
	at     map[K]*list.Element
	rows   int
	// OnEvict, if set, hears of each drawing let go.
	OnEvict func(K, V)
}

type entry[K comparable, V any] struct {
	k    K
	v    V
	rows int
}

// NewCache keeps up to budget rows of drawings past an Evict.
func NewCache[K comparable, V any](budget int) *Cache[K, V] {
	return &Cache[K, V]{budget: budget, used: list.New(), at: map[K]*list.Element{}}
}

// Get is k's drawing, which counts as using it.
func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	e := c.at[k]
	if e == nil {
		return v, false
	}
	c.used.MoveToFront(e)
	return e.Value.(*entry[K, V]).v, true
}

// Put keeps v as k's drawing, rows high.
func (c *Cache[K, V]) Put(k K, v V, rows int) {
	if e := c.at[k]; e != nil {
		en := e.Value.(*entry[K, V])
		c.rows += rows - en.rows
		en.v, en.rows = v, rows
		c.used.MoveToFront(e)
		return
	}
	c.at[k] = c.used.PushFront(&entry[K, V]{k, v, rows})
	c.rows += rows
}

// Evict lets the least recently used drawings go until the rest fit the
// budget, stopping at one keep says to hold (when keep is set): a
// drawing in the window just drawn, so all those used since are too.
func (c *Cache[K, V]) Evict(keep func(K) bool) {
	for c.rows > c.budget && c.used.Len() > 0 {
		e := c.used.Back()
		en := e.Value.(*entry[K, V])
		if keep != nil && keep(en.k) {
			return
		}
		c.used.Remove(e)
		delete(c.at, en.k)
		c.rows -= en.rows
		if c.OnEvict != nil {
			c.OnEvict(en.k, en.v)
		}
	}
}

// Each calls f with every drawing kept, most recently used first, without
// counting as using them.
func (c *Cache[K, V]) Each(f func(k K, v V, rows int)) {
	for e := c.used.Front(); e != nil; e = e.Next() {
		en := e.Value.(*entry[K, V])
		f(en.k, en.v, en.rows)
	}
}

// Clear lets every drawing go.
func (c *Cache[K, V]) Clear() {
	if c.OnEvict != nil {
		for e := c.used.Front(); e != nil; e = e.Next() {
			en := e.Value.(*entry[K, V])
			c.OnEvict(en.k, en.v)
		}
	}
	c.used.Init()
	clear(c.at)
	c.rows = 0
}

// Rows is the rows of drawings kept; Len how many drawings.
func (c *Cache[K, V]) Rows() int { return c.rows }
func (c *Cache[K, V]) Len() int  { return len(c.at) }
