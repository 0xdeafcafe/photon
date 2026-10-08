package fuzzy

import "testing"

func TestMatch(t *testing.T) {
	a, _, _ := Match("dev", "dev-alerts")
	b, _, _ := Match("dev", "random-devices")
	c, at, ok := Match("pa", "prod-alerts")
	if a <= b || !ok || c <= 0 || len(at) != 2 || at[0] != 0 || at[1] != 5 {
		t.Fatalf("scores %d %d %d, at %v", a, b, c, at)
	}
	if _, _, ok := Match("zz", "dev"); ok {
		t.Fatal("zz matched dev")
	}
	Match("stan", "İstanbul") // lowering İ changes the length: mustn't panic
}
