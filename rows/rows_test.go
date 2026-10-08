package rows

import (
	"math/rand/v2"
	"testing"
)

// Index agrees with summing by hand, through Reset and Set.
func TestIndex(t *testing.T) {
	var ix Index
	if ix.Find(3) != 0 || ix.Total() != 0 {
		t.Fatal("empty index")
	}
	r := rand.New(rand.NewPCG(1, 2))
	for _, n := range []int{1, 2, 7, 64, 100} {
		h := make([]int, n)
		for i := range h {
			h[i] = r.IntN(5) // zero-height items happen (folded, deleted)
		}
		ix.Reset(h)
		for k := 0; k < 50; k++ {
			i := r.IntN(n)
			h[i] = r.IntN(9)
			ix.Set(i, h[i])
		}
		sum := 0
		for i := range h {
			if ix.Prefix(i) != sum {
				t.Fatalf("n %d: Prefix(%d) = %d, want %d", n, i, ix.Prefix(i), sum)
			}
			for row := sum; row < sum+h[i]; row++ {
				if ix.Find(row) != i {
					t.Fatalf("n %d: Find(%d) = %d, want %d", n, row, ix.Find(row), i)
				}
			}
			sum += h[i]
		}
		if ix.Total() != sum || ix.Find(sum+10) != n-1 || ix.Find(-1) != 0 {
			t.Fatalf("n %d: ends wrong", n)
		}
	}
}

func TestCache(t *testing.T) {
	c := NewCache[string, int](10)
	var gone []string
	c.OnEvict = func(k string, _ int) { gone = append(gone, k) }
	c.Put("a", 1, 4)
	c.Put("b", 2, 4)
	c.Put("c", 3, 4)
	c.Get("a") // a is now newer than b
	c.Evict(nil)
	if _, ok := c.Get("b"); ok || c.Rows() != 8 || len(gone) != 1 || gone[0] != "b" {
		t.Fatalf("evicted %v, rows %d", gone, c.Rows())
	}
	c.Put("d", 4, 6)
	c.Evict(func(k string) bool { return k == "c" }) // c is in the window
	if c.Len() != 3 || c.Rows() != 14 {
		t.Fatalf("keep should stop eviction: len %d rows %d", c.Len(), c.Rows())
	}
	c.Put("d", 5, 1) // replacing changes its cost
	if v, _ := c.Get("d"); v != 5 || c.Rows() != 9 {
		t.Fatalf("replace: %d rows", c.Rows())
	}
	var order string
	c.Each(func(k string, _ int, _ int) { order += k })
	if order != "dac" {
		t.Fatalf("each, newest first: %q", order)
	}
	c.Clear()
	if c.Len() != 0 || c.Rows() != 0 || len(gone) != 4 {
		t.Fatalf("clear: %v", gone)
	}
}

func BenchmarkFind(b *testing.B) {
	h := make([]int, 10000)
	for i := range h {
		h[i] = 3
	}
	var ix Index
	ix.Reset(h)
	for b.Loop() {
		ix.Find(17771)
	}
}
