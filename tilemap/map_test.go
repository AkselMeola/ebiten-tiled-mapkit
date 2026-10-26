package tilemap

import (
	"math"
	"testing"
	"time"
)

func TestFrameAt(t *testing.T) {
	var def TileDef
	def.SetAnimation([]Frame{
		{Tile: 4, Duration: 200 * time.Millisecond},
		{Tile: 5, Duration: 200 * time.Millisecond},
		{Tile: 6, Duration: 200 * time.Millisecond},
	})

	cases := map[time.Duration]TileID{
		0:                      4,
		250 * time.Millisecond: 5,
		500 * time.Millisecond: 6,
		650 * time.Millisecond: 4,
	}
	for clock, want := range cases {
		if got := def.FrameAt(clock); got != want {
			t.Errorf("FrameAt(%v) = %d, want %d", clock, got, want)
		}
	}
}

func TestFrameAtWithoutSetAnimation(t *testing.T) {
	def := TileDef{Animation: []Frame{{Tile: 1, Duration: time.Second}, {Tile: 2, Duration: time.Second}}}
	if got := def.FrameAt(1500 * time.Millisecond); got != 2 {
		t.Errorf("FrameAt = %d, want 2", got)
	}
}

func TestFrameAtStatic(t *testing.T) {
	var def TileDef
	if got := def.FrameAt(time.Second); got != 0 {
		t.Errorf("FrameAt = %d, want 0", got)
	}
}

func TestLayerLookup(t *testing.T) {
	inner := &TileLayer{LayerBase: LayerBase{Name: "inner"}}
	m := Map{Layers: []Layer{
		&ObjectLayer{LayerBase: LayerBase{Name: "objects"}},
		&GroupLayer{LayerBase: LayerBase{Name: "group"}, Layers: []Layer{inner}},
	}}
	if got := m.Layer("inner"); got != inner {
		t.Errorf("Layer(inner) = %v", got)
	}
	if got := m.Layer("missing"); got != nil {
		t.Errorf("Layer(missing) = %v", got)
	}
}

func TestObjectBounds(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	cases := []struct {
		name                   string
		o                      MapObject
		minX, minY, maxX, maxY float64
	}{
		{"rect", MapObject{X: 10, Y: 20, W: 16, H: 8}, 10, 20, 26, 28},
		{"rotated 90", MapObject{X: 10, Y: 20, W: 16, H: 8, Rotation: 90}, 2, 20, 10, 36},
		{"polygon", MapObject{X: 10, Y: 20, Shape: Shape{Kind: ShapePolygon, Points: []Point{{0, 0}, {4, -2}, {-1, 6}}}}, 9, 18, 14, 26},
		{"point", MapObject{X: 3, Y: 4, Shape: Shape{Kind: ShapePoint}}, 3, 4, 3, 4},
	}
	for _, tc := range cases {
		minX, minY, maxX, maxY := tc.o.Bounds()
		if !near(minX, tc.minX) || !near(minY, tc.minY) || !near(maxX, tc.maxX) || !near(maxY, tc.maxY) {
			t.Errorf("%s: Bounds = %v,%v,%v,%v, want %v,%v,%v,%v", tc.name, minX, minY, maxX, maxY, tc.minX, tc.minY, tc.maxX, tc.maxY)
		}
		if got := tc.o.SortY(); !near(got, tc.maxY) {
			t.Errorf("%s: SortY = %v, want %v", tc.name, got, tc.maxY)
		}
	}
}

func TestFrameAtOnce(t *testing.T) {
	var def TileDef
	def.SetAnimation([]Frame{{Tile: 4, Duration: 100 * time.Millisecond}, {Tile: 5, Duration: 100 * time.Millisecond}})

	cases := []struct {
		t    time.Duration
		id   TileID
		done bool
	}{
		{-time.Second, 4, false},
		{0, 4, false},
		{150 * time.Millisecond, 5, false},
		{200 * time.Millisecond, 5, true},
		{time.Hour, 5, true},
	}
	for _, tc := range cases {
		if id, done := def.FrameAtOnce(tc.t); id != tc.id || done != tc.done {
			t.Errorf("FrameAtOnce(%v) = %d,%v, want %d,%v", tc.t, id, done, tc.id, tc.done)
		}
	}

	var static TileDef
	if id, done := static.FrameAtOnce(time.Second); id != 0 || !done {
		t.Errorf("static FrameAtOnce = %d,%v", id, done)
	}
}
