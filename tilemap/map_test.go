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
	cases := []struct {
		name     string
		o        MapObject
		min, max Vector
	}{
		{"rect", MapObject{Pos: Vector{10, 20}, Size: Vector{16, 8}}, Vector{10, 20}, Vector{26, 28}},
		{"rotated 90", MapObject{Pos: Vector{10, 20}, Size: Vector{16, 8}, Rotation: 90}, Vector{2, 20}, Vector{10, 36}},
		{"polygon", MapObject{Pos: Vector{10, 20}, Shape: Shape{Kind: ShapePolygon, Points: []Vector{{0, 0}, {4, -2}, {-1, 6}}}}, Vector{9, 18}, Vector{14, 26}},
		{"point", MapObject{Pos: Vector{3, 4}, Shape: Shape{Kind: ShapePoint}}, Vector{3, 4}, Vector{3, 4}},
	}
	for _, tc := range cases {
		gotMin, gotMax := tc.o.Bounds()
		if !nearVec(gotMin, tc.min) || !nearVec(gotMax, tc.max) {
			t.Errorf("%s: Bounds = %v,%v, want %v,%v", tc.name, gotMin, gotMax, tc.min, tc.max)
		}
		if got := tc.o.SortY(); !near(got, tc.max.Y) {
			t.Errorf("%s: SortY = %v, want %v", tc.name, got, tc.max.Y)
		}
	}
}

func TestObjectPoints(t *testing.T) {
	o := MapObject{Pos: Vector{10, 20}, Size: Vector{16, 8}}
	if got := o.Center(); got != (Vector{18, 24}) {
		t.Errorf("Center = %v", got)
	}
	if got := o.Foot(); got != (Vector{18, 28}) {
		t.Errorf("Foot = %v", got)
	}
	o.Rotation = 90
	if got := o.Foot(); !nearVec(got, Vector{2, 28}) {
		t.Errorf("rotated Foot = %v", got)
	}
}

func TestVector(t *testing.T) {
	v := Vector{3, 4}
	if got := v.Add(Vector{1, 2}).Sub(Vector{2, 2}).Scale(2); got != (Vector{4, 8}) {
		t.Errorf("Add/Sub/Scale = %v", got)
	}
	if got := (Vector{1, 0}).Rotate(90); !nearVec(got, Vector{0, 1}) {
		t.Errorf("Rotate = %v", got)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func nearVec(a, b Vector) bool { return near(a.X, b.X) && near(a.Y, b.Y) }

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
