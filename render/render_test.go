package render

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

func TestDrawOrder(t *testing.T) {
	tileAt := func(y float64) tilemap.MapObject {
		return tilemap.MapObject{Pos: tilemap.Vector{Y: y}, Size: tilemap.Vector{X: 16, Y: 16}, Visible: true, Cell: tilemap.Cell{TileID: 1}}
	}
	hidden := tileAt(0)
	hidden.Visible = false
	plain := tilemap.MapObject{Visible: true}
	l := &tilemap.ObjectLayer{Objects: []tilemap.MapObject{tileAt(50), hidden, tileAt(10), plain, tileAt(50), tileAt(30)}}

	indexes := func(items []drawItem) []int {
		out := make([]int, len(items))
		for i, item := range items {
			out[i] = item.index
		}
		return out
	}

	if got := indexes(drawOrder(l, orthogonal{}, nil)); !slices.Equal(got, []int{2, 5, 0, 4}) {
		t.Errorf("topdown order = %v", got)
	}
	l.DrawOrder = tilemap.DrawOrderIndex
	if got := indexes(drawOrder(l, orthogonal{}, nil)); !slices.Equal(got, []int{0, 2, 4, 5}) {
		t.Errorf("index order = %v", got)
	}
}

type testSprite tilemap.Vector

func (s testSprite) Foot() tilemap.Vector { return tilemap.Vector(s) }

func (s testSprite) Draw(*ebiten.Image, float64, float64) {}

func TestSpriteOrder(t *testing.T) {
	// A 3x3 layer with a tree in the middle; sprites stand on tile centers around it.
	l := &tilemap.TileLayer{Width: 3, Height: 3, Cells: make([]tilemap.Cell, 9)}
	tree := tilemap.TilePos{Col: 1, Row: 1}
	l.Cells[l.Index(tree)] = tilemap.Cell{TileID: 1}

	for _, orientation := range []tilemap.Orientation{tilemap.OrientationOrthogonal, tilemap.OrientationIsometric} {
		m := &tilemap.Map{Width: 3, Height: 3, TileWidth: 32, TileHeight: 16, Orientation: orientation}
		proj, _ := newProjection(m)
		r := &MapRenderer{m: m, proj: proj}
		standOn := func(p tilemap.TilePos) testSprite {
			return testSprite(m.TileToWorld(p).Add(tilemap.Vector{X: 16, Y: 8}))
		}

		cases := []struct {
			stand  tilemap.TilePos
			behind bool
		}{
			{tree.Add(tilemap.Up), true},
			{tree.Add(tilemap.Left), orientation == tilemap.OrientationIsometric},
			{tree.Add(tilemap.Down), false},
			{tree.Add(tilemap.Right), false},
		}
		for _, tc := range cases {
			items := r.tileOrder(l, image.Rect(0, 0, 3, 3), []Sprite{standOn(tc.stand)}, nil)
			if behind := items[0].sprite; behind != tc.behind {
				t.Errorf("orientation %d: sprite on %v behind the tree = %v, want %v", orientation, tc.stand, behind, tc.behind)
			}
		}
	}
}

func TestOrthogonal(t *testing.T) {
	m := &tilemap.Map{Width: 10, Height: 5, TileWidth: 16, TileHeight: 8}
	p := newOrthogonal(m)

	if x, y := p.tileToScreen(tilemap.TilePos{Col: 3, Row: 2}); x != 48 || y != 16 {
		t.Errorf("tileToScreen = %v,%v", x, y)
	}

	cases := []struct {
		view           image.Rectangle
		c0, r0, c1, r1 int
	}{
		{image.Rect(0, 0, 32, 16), 0, 0, 1, 1},
		{image.Rect(10, 4, 33, 17), 0, 0, 2, 2},
		{image.Rect(-100, -100, 1000, 1000), 0, 0, 9, 4},
		{image.Rect(-50, -50, 0, 0), 0, 0, -1, -1},
	}
	for _, tc := range cases {
		c0, r0, c1, r1 := p.visibleTiles(tc.view)
		if c0 != tc.c0 || r0 != tc.r0 || c1 != tc.c1 || r1 != tc.r1 {
			t.Errorf("visibleTiles(%v) = %d,%d,%d,%d, want %d,%d,%d,%d", tc.view, c0, r0, c1, r1, tc.c0, tc.r0, tc.c1, tc.r1)
		}
	}
}

func TestIsometric(t *testing.T) {
	// 64x32 tiles on a 4x3 map: the map's image is (4+3)*32 wide, tile (0,0) is the top diamond.
	m := &tilemap.Map{Width: 4, Height: 3, TileWidth: 64, TileHeight: 32, Orientation: tilemap.OrientationIsometric}
	p := newIsometric(m)

	if x, y := p.tileToScreen(tilemap.TilePos{}); x != 64 || y != 0 {
		t.Errorf("tileToScreen(0,0) = %v,%v, want 64,0", x, y)
	}
	if x, y := p.tileToScreen(tilemap.TilePos{Col: 3, Row: 2}); x != 96 || y != 80 {
		t.Errorf("tileToScreen(3,2) = %v,%v, want 96,80", x, y)
	}
	// The top corner of tile (1,0) and the center of tile (0,0).
	if x, y := p.worldToScreen(tilemap.Vector{X: 64}); x != 128 || y != 16 {
		t.Errorf("worldToScreen(64,0) = %v,%v", x, y)
	}
	if x, y := p.worldToScreen(tilemap.Vector{X: 32, Y: 16}); x != 96 || y != 16 {
		t.Errorf("worldToScreen(32,16) = %v,%v", x, y)
	}
	for _, v := range []tilemap.Vector{{}, {X: 64}, {X: 100, Y: 50}, {X: -10, Y: 300}} {
		x, y := p.worldToScreen(v)
		if got := p.screenToWorld(x, y); !near(got.X, v.X) || !near(got.Y, v.Y) {
			t.Errorf("screenToWorld(worldToScreen(%v)) = %v", v, got)
		}
	}

	// Every tile whose diamond center is in view is in the range.
	view := image.Rect(100, 20, 180, 60)
	c0, r0, c1, r1 := p.visibleTiles(view)
	for row := range m.Height {
		for col := range m.Width {
			x, y := p.worldToScreen(tilemap.Vector{X: (float64(col) + 0.5) * 64, Y: (float64(row) + 0.5) * 32})
			in := image.Pt(int(x), int(y)).In(view)
			if in && (col < c0 || col > c1 || row < r0 || row > r1) {
				t.Errorf("tile %d,%d in view but outside %d,%d..%d,%d", col, row, c0, r0, c1, r1)
			}
		}
	}

	// Tile objects are as deep as the tile whose bottom corner is their foot: (2,0) and (0,2) are
	// level, (1,2) is in front.
	foot := func(col, row float64) *tilemap.MapObject {
		return &tilemap.MapObject{Pos: tilemap.Vector{X: col*64 - 32, Y: row*32 - 10}, Size: tilemap.Vector{X: 64, Y: 10}}
	}
	if a, b, c := p.objectDepth(foot(2, 0)), p.objectDepth(foot(0, 2)), p.objectDepth(foot(1, 2)); a != b || c <= a {
		t.Errorf("depths = %v, %v, %v", a, b, c)
	}
}

func TestApplyFlips(t *testing.T) {
	// A 16x8 image; track where its top-left corner and drawn size end up.
	bounds := image.Rect(0, 0, 16, 8)
	cases := []struct {
		flip             tilemap.Flip
		w, h             float64
		cornerX, cornerY float64
	}{
		{0, 16, 8, 0, 0},
		{tilemap.FlipH, 16, 8, 16, 0},
		{tilemap.FlipV, 16, 8, 0, 8},
		{tilemap.FlipH | tilemap.FlipV, 16, 8, 16, 8},
		{tilemap.FlipD, 8, 16, 0, 0},
		{tilemap.FlipD | tilemap.FlipH, 8, 16, 8, 0},
	}
	for _, tc := range cases {
		var g ebiten.GeoM
		w, h := applyFlips(&g, tc.flip, bounds)
		if w != tc.w || h != tc.h {
			t.Errorf("flip %b size = %vx%v, want %vx%v", tc.flip, w, h, tc.w, tc.h)
		}
		x, y := g.Apply(0, 0)
		if !near(x, tc.cornerX) || !near(y, tc.cornerY) {
			t.Errorf("flip %b corner = %v,%v, want %v,%v", tc.flip, x, y, tc.cornerX, tc.cornerY)
		}
		// The whole image stays within [0,w]x[0,h].
		for _, p := range [][2]float64{{0, 0}, {16, 0}, {0, 8}, {16, 8}} {
			x, y := g.Apply(p[0], p[1])
			if x < -1e-9 || y < -1e-9 || x > w+1e-9 || y > h+1e-9 {
				t.Errorf("flip %b maps %v to %v,%v outside %vx%v", tc.flip, p, x, y, w, h)
			}
		}
	}
}

func TestOverdraw(t *testing.T) {
	m := &tilemap.Map{
		TileWidth: 16, TileHeight: 16,
		Tiles: []tilemap.TileDef{
			{},
			{Src: image.Rect(0, 0, 16, 16)},
			{Src: image.Rect(0, 0, 16, 40), Offset: image.Pt(-2, 3)},
		},
	}
	want := image.Rectangle{Min: image.Pt(2, 21), Max: image.Pt(22, 3)}
	if got := overdraw(m); got != want {
		t.Errorf("overdraw = %v, want %v", got, want)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestWorldScreen(t *testing.T) {
	r := &MapRenderer{proj: orthogonal{}}
	cam := Camera{X: 15.6, Y: -0.4}

	if x, y := r.WorldToScreen(tilemap.Vector{X: 100, Y: 50.5}, cam); x != 84 || y != 50.5 {
		t.Errorf("WorldToScreen = %v,%v, want 84,50.5: camera rounded, position kept", x, y)
	}
	if got := r.ScreenToWorld(20, 5, cam); got != (tilemap.Vector{X: 36, Y: 5}) {
		t.Errorf("ScreenToWorld = %v", got)
	}
	world := tilemap.Vector{X: 7, Y: 9}
	if x, y := r.WorldToScreen(world, cam); r.ScreenToWorld(int(x), int(y), cam) != world {
		t.Error("ScreenToWorld does not invert WorldToScreen")
	}
}

func TestNewMapRenderer(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 32, 16))); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"tiles.png": {Data: buf.Bytes()}}
	tile := func(src image.Rectangle) tilemap.TileDef { return tilemap.TileDef{Image: "tiles.png", Src: src} }
	m := &tilemap.Map{Width: 2, Height: 1, TileWidth: 16, TileHeight: 16, Tiles: []tilemap.TileDef{
		{}, tile(image.Rect(0, 0, 16, 16)), tile(image.Rect(16, 0, 32, 16)),
	}}

	r, err := NewMapRenderer(NewImageCache(fsys), m)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.tiles[2].Bounds(); got != image.Rect(16, 0, 32, 16) {
		t.Errorf("tile 2 bounds = %v", got)
	}

	m.Layers = []tilemap.Layer{&tilemap.GroupLayer{Layers: []tilemap.Layer{&tilemap.ImageLayer{Image: "sky.png"}}}}
	if _, err := NewMapRenderer(NewImageCache(fsys), m); err == nil || !strings.Contains(err.Error(), "sky.png") {
		t.Errorf("error = %v, want one naming sky.png", err)
	}

	m.Layers = nil
	m.Orientation = tilemap.OrientationStaggered
	if _, err := NewMapRenderer(NewImageCache(fsys), m); err == nil {
		t.Error("staggered map: no error")
	}
}
