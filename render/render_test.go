package render

import (
	"image"
	"math"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

func TestDrawOrder(t *testing.T) {
	tileAt := func(y float64) tilemap.MapObject {
		return tilemap.MapObject{Pos: tilemap.Vector{Y: y}, Size: tilemap.Vector{X: 16, Y: 16}, Visible: true, Tile: tilemap.Cell{Tile: 1}}
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

	if got := indexes(drawOrder(l, nil)); !slices.Equal(got, []int{2, 5, 0, 4}) {
		t.Errorf("topdown order = %v", got)
	}
	l.DrawOrder = tilemap.DrawOrderIndex
	if got := indexes(drawOrder(l, nil)); !slices.Equal(got, []int{0, 2, 4, 5}) {
		t.Errorf("index order = %v", got)
	}
}

func TestOrthogonal(t *testing.T) {
	m := &tilemap.Map{Width: 10, Height: 5, TileWidth: 16, TileHeight: 8}
	p := NewOrthogonal(m)

	if x, y := p.TileToScreen(3, 2); x != 48 || y != 16 {
		t.Errorf("TileToScreen = %v,%v", x, y)
	}
	if c, r := p.ScreenToTile(47.9, 16); c != 2 || r != 2 {
		t.Errorf("ScreenToTile = %d,%d", c, r)
	}
	if c, r := p.ScreenToTile(-0.5, -9); c != -1 || r != -2 {
		t.Errorf("ScreenToTile negative = %d,%d", c, r)
	}
	if b := p.Bounds(m); b != image.Rect(0, 0, 160, 40) {
		t.Errorf("Bounds = %v", b)
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
		c0, r0, c1, r1 := p.VisibleTiles(m, tc.view)
		if c0 != tc.c0 || r0 != tc.r0 || c1 != tc.c1 || r1 != tc.r1 {
			t.Errorf("VisibleTiles(%v) = %d,%d,%d,%d, want %d,%d,%d,%d", tc.view, c0, r0, c1, r1, tc.c0, tc.r0, tc.c1, tc.r1)
		}
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
