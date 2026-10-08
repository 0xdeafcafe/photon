package frame

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestGate(t *testing.T) {
	var g Gate
	draws := 0
	draw := func() string { draws++; return "frame" }

	g.Frame(draw)
	g.Keep()
	if g.Frame(draw); draws != 1 {
		t.Fatalf("Keep should reuse the last frame: %d draws", draws)
	}
	g.Frame(draw)
	if draws != 2 {
		t.Fatal("without Keep the next frame is drawn")
	}

	// Turns straight after a frame are held, and only one tick is asked for.
	wheel := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	if g.Wheel(wheel) == nil || g.Wheel(wheel) != nil {
		t.Fatal("want one tick for a burst")
	}
	if g.Frame(draw); draws != 2 {
		t.Fatal("a held turn drew")
	}
	if !g.Tick(TickMsg{}) || g.Tick(wheel) {
		t.Fatal("Tick should know its own message only")
	}
	if g.Frame(draw); draws != 3 {
		t.Fatal("the tick should draw")
	}
	if g.Wheel(tea.KeyPressMsg{}) != nil {
		t.Fatal("only the wheel is held")
	}
}
