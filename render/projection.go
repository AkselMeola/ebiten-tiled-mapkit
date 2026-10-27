// Package render draws tilemap.Map with Ebiten.
package render

import (
	"image"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// projection places tiles for one map orientation. Placement and culling go through it, so other
// orientations only add an implementation.
type projection interface {
	// tileToWorld is p's top-left corner in world pixels.
	tileToWorld(p tilemap.TilePos) (x, y float64)
	// visibleTiles returns the inclusive tile range covering view; empty when c0 > c1 or r0 > r1.
	visibleTiles(m *tilemap.Map, view image.Rectangle) (c0, r0, c1, r1 int)
}

type orthogonal struct {
	tileWidth, tileHeight int
}

func newOrthogonal(m *tilemap.Map) orthogonal {
	return orthogonal{tileWidth: m.TileWidth, tileHeight: m.TileHeight}
}

func (o orthogonal) tileToWorld(p tilemap.TilePos) (x, y float64) {
	return float64(p.Col * o.tileWidth), float64(p.Row * o.tileHeight)
}

func (o orthogonal) visibleTiles(m *tilemap.Map, view image.Rectangle) (c0, r0, c1, r1 int) {
	c0 = max(0, floorDiv(view.Min.X, o.tileWidth))
	r0 = max(0, floorDiv(view.Min.Y, o.tileHeight))
	c1 = min(m.Width-1, floorDiv(view.Max.X-1, o.tileWidth))
	r1 = min(m.Height-1, floorDiv(view.Max.Y-1, o.tileHeight))
	return c0, r0, c1, r1
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
