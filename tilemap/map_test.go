package tilemap

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestFrame(t *testing.T) {
	m := &Map{Tiles: make([]TileDef, 7)}
	m.Tiles[1].Animation = []Frame{
		{TileID: 4, Duration: 200 * time.Millisecond},
		{TileID: 5, Duration: 200 * time.Millisecond},
		{TileID: 6, Duration: 200 * time.Millisecond},
	}

	cases := map[time.Duration]TileID{
		0:                      4,
		250 * time.Millisecond: 5,
		500 * time.Millisecond: 6,
		650 * time.Millisecond: 4,
	}
	for clock, want := range cases {
		if got := m.Frame(1, clock); got != want {
			t.Errorf("Frame(1, %v) = %d, want %d", clock, got, want)
		}
	}
	for _, id := range []TileID{0, 2, 99} {
		if got := m.Frame(id, time.Second); got != id {
			t.Errorf("Frame(%d) = %d, want itself", id, got)
		}
	}
}

func TestTileWhere(t *testing.T) {
	m := &Map{Tiles: make([]TileDef, 4)}
	m.Tiles[0].Props = Properties{"name": "idle"} // empty slot, never matched
	m.Tiles[1].Props = Properties{"name": "idle", "dir": "left", "speed": 2}
	m.Tiles[2].Props = Properties{"name": "run", "dir": "left", "nested": Properties{"a": 1}}
	m.Tiles[3].Props = Properties{"name": "run", "dir": "right"}

	cases := []struct {
		name  string
		value any
		want  TileID
	}{
		{"name", "run", 2},
		{"speed", 2, 1},
		{"speed", 2.0, 0},
		{"name", "jump", 0},
		{"missing", "run", 0},
		{"nested", Properties{"a": 1}, 0},
	}
	for _, c := range cases {
		if got := m.TileWhereProp(c.name, c.value); got != c.want {
			t.Errorf("TileWhereProp(%q, %v) = %d, want %d", c.name, c.value, got, c.want)
		}
	}

	got := m.TileWhereFunc(func(def *TileDef) bool {
		return def.Props.String("name") == "run" && def.Props.String("dir") == "right"
	})
	if got != 3 {
		t.Errorf("TileWhereFunc(run, right) = %d, want 3", got)
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

func TestFrameOnce(t *testing.T) {
	m := &Map{Tiles: make([]TileDef, 6)}
	m.Tiles[1].Animation = []Frame{{TileID: 4, Duration: 100 * time.Millisecond}, {TileID: 5, Duration: 100 * time.Millisecond}}

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
		if id, done := m.FrameOnce(1, tc.t); id != tc.id || done != tc.done {
			t.Errorf("FrameOnce(1, %v) = %d,%v, want %d,%v", tc.t, id, done, tc.id, tc.done)
		}
	}

	if id, done := m.FrameOnce(2, time.Second); id != 2 || !done {
		t.Errorf("static FrameOnce = %d,%v", id, done)
	}
}

func TestObjects(t *testing.T) {
	m := &Map{Layers: []Layer{
		&ObjectLayer{LayerBase: LayerBase{Name: "a"}, Objects: []MapObject{{ID: 1}, {ID: 2}}},
		&TileLayer{LayerBase: LayerBase{Name: "ground"}},
		&GroupLayer{LayerBase: LayerBase{Name: "g"}, Layers: []Layer{
			&ObjectLayer{LayerBase: LayerBase{Name: "b"}, Objects: []MapObject{{ID: 3}}},
			&TileLayer{LayerBase: LayerBase{Name: "top"}},
		}},
	}}

	var names []string
	for l := range m.AllLayers() {
		names = append(names, l.Base().Name)
	}
	if got := strings.Join(names, ","); got != "a,ground,g,b,top" {
		t.Errorf("AllLayers = %s", got)
	}

	var tileLayers []string
	for l := range m.TileLayers() {
		tileLayers = append(tileLayers, l.Name)
	}
	if got := strings.Join(tileLayers, ","); got != "ground,top" {
		t.Errorf("TileLayers = %s", got)
	}

	var ids []int
	for o := range m.Objects() {
		ids = append(ids, o.ID)
		o.Visible = true
	}
	if !slices.Equal(ids, []int{1, 2, 3}) {
		t.Errorf("Objects = %v", ids)
	}
	if !m.Layer("b").(*ObjectLayer).Objects[0].Visible {
		t.Error("change through Objects did not stick")
	}
	if o := m.Object(3); o == nil || o.ID != 3 {
		t.Errorf("Object(3) = %v", o)
	}
	if o := m.Object(9); o != nil {
		t.Errorf("Object(9) = %v, want nil", o)
	}

	for range m.Objects() {
		break
	}
	for range m.TileLayers() {
		break
	}
}

func TestTilePositions(t *testing.T) {
	l := &TileLayer{Width: 3, Height: 2, Cells: []Cell{{TileID: 1}, {}, {TileID: 2}, {}, {TileID: 3}, {}}}
	p := TilePos{Col: 1, Row: 1}

	if got := p.Add(Right).Add(Up); got != (TilePos{Col: 2, Row: 0}) {
		t.Errorf("Add = %v", got)
	}
	if got := l.At(p); got.TileID != 3 {
		t.Errorf("At = %v", got)
	}
	if got := l.At(p.Add(Down)); !got.Empty() {
		t.Errorf("At outside = %v", got)
	}
	if got := l.Index(p); got != 4 {
		t.Errorf("Index = %d", got)
	}
	if got := l.Index(TilePos{Col: 3, Row: 0}); got != -1 {
		t.Errorf("Index outside = %d", got)
	}
	if got := l.Pos(4); got != p {
		t.Errorf("Pos = %v", got)
	}

	var cells []string
	for pos, c := range l.All() {
		cells = append(cells, fmt.Sprintf("%d,%d:%d", pos.Col, pos.Row, c.TileID))
	}
	if got := strings.Join(cells, " "); got != "0,0:1 2,0:2 1,1:3" {
		t.Errorf("All = %s", got)
	}
	for range l.All() {
		break
	}

	m := &Map{Width: 3, Height: 2}
	if !m.InBounds(p) || m.InBounds(p.Add(Down)) || m.InBounds(TilePos{Col: -1}) {
		t.Error("InBounds")
	}
}

func TestObjectsAt(t *testing.T) {
	tile := func(id, x, y, w, h float64) MapObject {
		return MapObject{ID: int(id), Pos: Vector{x, y}, Size: Vector{w, h}, Visible: true}
	}
	hidden := tile(4, 32, 0, 16, 16)
	hidden.Visible = false
	m := &Map{Width: 4, Height: 4, TileWidth: 16, TileHeight: 16, Layers: []Layer{
		&ObjectLayer{Objects: []MapObject{
			tile(1, 15.5, 0, 16, 16), // slightly off the grid: covers tile 1,0 only
			tile(2, 0, 16, 32, 32),   // 2x2
			tile(3, 40, 40, 0, 0),    // point
			hidden,
		}},
	}}
	ids := func(pos TilePos) []int {
		var out []int
		for o := range m.ObjectsAt(pos) {
			out = append(out, o.ID)
		}
		return out
	}

	cases := []struct {
		pos  TilePos
		want []int
	}{
		{TilePos{0, 0}, nil},
		{TilePos{1, 0}, []int{1}},
		{TilePos{0, 1}, []int{2}},
		{TilePos{1, 2}, []int{2}},
		{TilePos{2, 1}, nil},
		{TilePos{2, 2}, []int{3}},
		{TilePos{2, 0}, nil},
	}
	for _, tc := range cases {
		if got := ids(tc.pos); !slices.Equal(got, tc.want) {
			t.Errorf("ObjectsAt(%v) = %v, want %v", tc.pos, got, tc.want)
		}
	}

	// On an isometric map a tile object is on the tile whose bottom corner is nearest its foot.
	m.Orientation = OrientationIsometric
	m.Layers = []Layer{&ObjectLayer{Objects: []MapObject{{ID: 5, Pos: Vector{24, 4}, Size: Vector{16, 40}, Visible: true, Cell: Cell{TileID: 1}}}}}
	if got := ids(TilePos{1, 2}); !slices.Equal(got, []int{5}) {
		t.Errorf("isometric ObjectsAt(1,2) = %v, want [5]", got)
	}
	if got := m.ObjectTile(&m.Layers[0].(*ObjectLayer).Objects[0]); got != (TilePos{1, 2}) {
		t.Errorf("isometric ObjectTile = %v, want 1,2", got)
	}
	m.Orientation = OrientationOrthogonal
	if got := m.ObjectTile(&m.Layers[0].(*ObjectLayer).Objects[0]); got != (TilePos{2, 1}) {
		t.Errorf("orthogonal ObjectTile = %v, want the center's tile 2,1", got)
	}
}

func TestWorldToTile(t *testing.T) {
	m := &Map{TileWidth: 16, TileHeight: 8}
	cases := []struct {
		world Vector
		want  TilePos
	}{
		{Vector{47.9, 16}, TilePos{2, 2}},
		{Vector{48, 15.9}, TilePos{3, 1}},
		{Vector{-0.5, -9}, TilePos{-1, -2}},
	}
	for _, tc := range cases {
		if got := m.WorldToTile(tc.world); got != tc.want {
			t.Errorf("WorldToTile(%v) = %v, want %v", tc.world, got, tc.want)
		}
	}
	if got := m.TileToWorld(TilePos{3, 2}); got != (Vector{48, 16}) {
		t.Errorf("TileToWorld = %v", got)
	}
	if got := m.WorldToTile(m.TileToWorld(TilePos{-2, 5})); got != (TilePos{-2, 5}) {
		t.Errorf("round trip = %v", got)
	}
}
