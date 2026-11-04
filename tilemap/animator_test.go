package tilemap

import (
	"testing"
	"time"
)

// animatorMap has tile 1 animated as 4, 5 at 100ms each, and tiles 2-5 static.
func animatorMap() *Map {
	m := &Map{Tiles: make([]TileDef, 6)}
	m.Tiles[1].Animation = []Frame{{TileID: 4, Duration: 100 * time.Millisecond}, {TileID: 5, Duration: 100 * time.Millisecond}}
	return m
}

// step advances a by 50ms and returns the frame and Done.
func step(a *Animator) (TileID, bool) {
	a.Update(50 * time.Millisecond)
	return a.Frame(), a.Done()
}

func TestAnimatorLoops(t *testing.T) {
	a := NewAnimator(animatorMap())
	a.Play(1)
	want := []TileID{4, 5, 5, 4, 4, 5, 5, 4}
	for i, w := range want {
		if f, done := step(a); f != w || done {
			t.Fatalf("step %d: frame %d done %v, want %d false", i+1, f, done, w)
		}
	}
}

func TestAnimatorPlayTimes(t *testing.T) {
	a := NewAnimator(animatorMap())
	a.PlayTimes(1, 3, 2)
	if a.Frame() != 4 {
		t.Fatalf("first frame %d, want 4", a.Frame())
	}
	// Three cycles of 200ms are 12 steps of 50ms; Done comes on the 12th.
	want := []TileID{4, 5, 5, 4, 4, 5, 5, 4, 4, 5, 5}
	for i, w := range want {
		if f, done := step(a); f != w || done {
			t.Fatalf("step %d: frame %d done %v, want %d false", i+1, f, done, w)
		}
	}
	if f, done := step(a); f != 2 || !done {
		t.Fatalf("step 12: frame %d done %v, want 2 true", f, done)
	}
	if f, done := step(a); f != 2 || done {
		t.Fatalf("after: frame %d done %v, want 2 false", f, done)
	}
	if a.Tile() != 2 {
		t.Errorf("Tile = %d, want 2", a.Tile())
	}
}

func TestAnimatorPlayOnce(t *testing.T) {
	a := NewAnimator(animatorMap())
	a.PlayOnce(1, 3)
	for range 3 {
		step(a)
	}
	if f, done := step(a); f != 3 || !done {
		t.Fatalf("frame %d done %v, want 3 true", f, done)
	}
}

func TestAnimatorPlayDoesNotRestart(t *testing.T) {
	a := NewAnimator(animatorMap())
	a.Play(1)
	step(a)
	step(a)
	a.Play(1)
	if a.Frame() != 5 {
		t.Errorf("Play of the looping tile restarted it: frame %d, want 5", a.Frame())
	}

	// PlayOnce always restarts, even on the same tile.
	a.PlayOnce(1, 2)
	if a.Frame() != 4 {
		t.Errorf("PlayOnce didn't restart: frame %d, want 4", a.Frame())
	}
}

func TestAnimatorStatic(t *testing.T) {
	a := NewAnimator(animatorMap())
	a.Play(2)
	if f, done := step(a); f != 2 || done {
		t.Errorf("static loop: frame %d done %v", f, done)
	}

	a.PlayOnce(3, 2)
	if a.Frame() != 3 {
		t.Errorf("static once before Update: frame %d, want 3", a.Frame())
	}
	if f, done := step(a); f != 2 || !done {
		t.Errorf("static once: frame %d done %v, want 2 true", f, done)
	}
}
