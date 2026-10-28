// Package render draws tilemap.Map with Ebiten.
package render

import (
	"fmt"
	"image"
	"math"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// projection maps world positions to the map's screen pixels, before the camera. Placement,
// culling and depth go through it, so other orientations only add an implementation.
type projection interface {
	// tileToScreen is the top-left of p's bounding box; tile images sit on its bottom-left.
	tileToScreen(p tilemap.TilePos) (x, y float64)
	worldToScreen(v tilemap.Vector) (x, y float64)
	screenToWorld(x, y float64) tilemap.Vector
	// visibleTiles returns the inclusive tile range covering view; empty when c0 > c1 or r0 > r1.
	visibleTiles(view image.Rectangle) (c0, r0, c1, r1 int)
	// depth orders what stands at world position v: higher is drawn later.
	depth(v tilemap.Vector) float64
	objectDepth(o *tilemap.MapObject) float64
}

func newProjection(m *tilemap.Map) (projection, error) {
	switch m.Orientation {
	case tilemap.OrientationOrthogonal, tilemap.OrientationUnknown:
		return newOrthogonal(m), nil
	case tilemap.OrientationIsometric:
		return newIsometric(m), nil
	}
	return nil, fmt.Errorf("render: orientation %d is not supported", m.Orientation)
}

type orthogonal struct {
	tileWidth, tileHeight int
	width, height         int
}

func newOrthogonal(m *tilemap.Map) orthogonal {
	return orthogonal{tileWidth: m.TileWidth, tileHeight: m.TileHeight, width: m.Width, height: m.Height}
}

func (o orthogonal) tileToScreen(p tilemap.TilePos) (x, y float64) {
	return float64(p.Col * o.tileWidth), float64(p.Row * o.tileHeight)
}

func (o orthogonal) worldToScreen(v tilemap.Vector) (x, y float64) { return v.X, v.Y }

func (o orthogonal) screenToWorld(x, y float64) tilemap.Vector { return tilemap.Vector{X: x, Y: y} }

func (o orthogonal) visibleTiles(view image.Rectangle) (c0, r0, c1, r1 int) {
	c0 = max(0, floorDiv(view.Min.X, o.tileWidth))
	r0 = max(0, floorDiv(view.Min.Y, o.tileHeight))
	c1 = min(o.width-1, floorDiv(view.Max.X-1, o.tileWidth))
	r1 = min(o.height-1, floorDiv(view.Max.Y-1, o.tileHeight))
	return c0, r0, c1, r1
}

func (o orthogonal) depth(v tilemap.Vector) float64 { return v.Y }

func (o orthogonal) objectDepth(obj *tilemap.MapObject) float64 { return obj.SortY() }

// isometric is Tiled's diamond layout: tile (0,0) is the top diamond and the map's left edge is
// at x 0. World positions are col × tileWidth, row × tileHeight, as on orthogonal maps.
type isometric struct {
	tileWidth, tileHeight float64
	width, height         int
}

func newIsometric(m *tilemap.Map) isometric {
	return isometric{tileWidth: float64(m.TileWidth), tileHeight: float64(m.TileHeight), width: m.Width, height: m.Height}
}

func (s isometric) tileToScreen(p tilemap.TilePos) (x, y float64) {
	x, y = s.worldToScreen(tilemap.Vector{X: float64(p.Col) * s.tileWidth, Y: float64(p.Row) * s.tileHeight})
	return x - s.tileWidth/2, y
}

func (s isometric) worldToScreen(v tilemap.Vector) (x, y float64) {
	col, row := v.X/s.tileWidth, v.Y/s.tileHeight
	return float64(s.height)*s.tileWidth/2 + (col-row)*s.tileWidth/2, (col + row) * s.tileHeight / 2
}

func (s isometric) screenToWorld(x, y float64) tilemap.Vector {
	diff := (x - float64(s.height)*s.tileWidth/2) / (s.tileWidth / 2)
	sum := y / (s.tileHeight / 2)
	return tilemap.Vector{X: (sum + diff) / 2 * s.tileWidth, Y: (sum - diff) / 2 * s.tileHeight}
}

// visibleTiles is the tile rectangle around the view's corners, one tile wider on each side for
// diamonds that only touch the view. It includes off-screen tiles, but the view is a diamond in
// tile space and a rectangle keeps tile iteration in row order.
func (s isometric) visibleTiles(view image.Rectangle) (c0, r0, c1, r1 int) {
	minCol, minRow := math.Inf(1), math.Inf(1)
	maxCol, maxRow := math.Inf(-1), math.Inf(-1)
	for _, corner := range [4]image.Point{view.Min, {view.Max.X, view.Min.Y}, {view.Min.X, view.Max.Y}, view.Max} {
		v := s.screenToWorld(float64(corner.X), float64(corner.Y))
		col, row := v.X/s.tileWidth, v.Y/s.tileHeight
		minCol, maxCol = min(minCol, col), max(maxCol, col)
		minRow, maxRow = min(minRow, row), max(maxRow, row)
	}
	c0 = max(0, int(math.Floor(minCol))-1)
	r0 = max(0, int(math.Floor(minRow))-1)
	c1 = min(s.width-1, int(math.Floor(maxCol))+1)
	r1 = min(s.height-1, int(math.Floor(maxRow))+1)
	return c0, r0, c1, r1
}

func (s isometric) depth(v tilemap.Vector) float64 {
	_, y := s.worldToScreen(v)
	return y
}

// objectDepth is that of the tile the object stands on: its foot is the tile's bottom corner.
func (s isometric) objectDepth(o *tilemap.MapObject) float64 {
	return s.depth(o.Foot()) - s.tileHeight/2
}

func floorDiv(a, b int) int {
	if b <= 0 {
		return 0
	}
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}
