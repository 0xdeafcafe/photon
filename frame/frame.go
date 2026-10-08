// Package frame decides when a bubbletea model draws a new frame: not
// after a message that changed nothing on screen, and not more than once
// a Gap while the wheel turns, with the frame the burst ends on drawn
// once it stops. It's rush's (ui/model.go holdWheelFrame, ui/view.go).
package frame

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Gap is the least time between frames the wheel draws: a trackpad sends
// turns far faster than anyone reads them.
const Gap = 16 * time.Millisecond

// TickMsg is the Gate's own tick, sent by what Wheel returns.
type TickMsg struct{}

// Gate holds a model's last frame. The zero value is ready.
type Gate struct {
	last    string
	same    bool // the next frame may be the last one again
	drawnAt time.Time
	pending bool // a TickMsg is on its way
}

// Keep says the message being handled changed nothing on screen.
func (g *Gate) Keep() { g.same = true }

// Tick says whether msg is the Gate's own tick, which draws what the
// wheel moved; Update has nothing else to do for it.
func (g *Gate) Tick(msg tea.Msg) bool {
	if _, ok := msg.(TickMsg); ok {
		g.pending = false
		return true
	}
	return false
}

// Wheel keeps the last frame for a wheel turn within Gap of it, and
// returns the tick that draws once the turns stop. Call it after
// handling msg, whatever msg is, and batch what it returns.
func (g *Gate) Wheel(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tea.MouseWheelMsg); !ok || g.same || time.Since(g.drawnAt) >= Gap {
		return nil
	}
	g.same = true
	if g.pending {
		return nil
	}
	g.pending = true
	return tea.Tick(Gap, func(time.Time) tea.Msg { return TickMsg{} })
}

// Frame is the frame to show: the last one if nothing's changed since,
// else what draw makes.
func (g *Gate) Frame(draw func() string) string {
	if !g.same || g.last == "" {
		g.last, g.drawnAt = draw(), time.Now()
	}
	g.same = false
	return g.last
}

// Last is the last frame drawn.
func (g *Gate) Last() string { return g.last }

// Held is whether the next frame may be the last one again, and whether a
// TickMsg is on its way.
func (g *Gate) Held() (same, pending bool) { return g.same, g.pending }

// DrawnAt says when the last frame was drawn, for a test that times a burst.
func (g *Gate) DrawnAt(t time.Time) { g.drawnAt = t }
