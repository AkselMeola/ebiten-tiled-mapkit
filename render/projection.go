// Package render draws tilemap.Map with Ebiten.
package render

import (
	"image"
	"math"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

type Projection interface {
	TileToScreen(col, row int) (x, y float64)
	ScreenToTile(x, y float64) (col, row int)
	Bounds(m *tilemap.Map) image.Rectangle
	// VisibleTiles returns the inclusive tile range covering view; empty when c0 > c1 or r0 > r1.
	VisibleTiles(m *tilemap.Map, view image.Rectangle) (c0, r0, c1, r1 int)
}

type Orthogonal struct {
	TileWidth, TileHeight int
}

func NewOrthogonal(m *tilemap.Map) Orthogonal {
	return Orthogonal{TileWidth: m.TileWidth, TileHeight: m.TileHeight}
}

func (o Orthogonal) TileToScreen(col, row int) (x, y float64) {
	return float64(col * o.TileWidth), float64(row * o.TileHeight)
}

func (o Orthogonal) ScreenToTile(x, y float64) (col, row int) {
	return floorDiv(int(math.Floor(x)), o.TileWidth), floorDiv(int(math.Floor(y)), o.TileHeight)
}

func (o Orthogonal) Bounds(m *tilemap.Map) image.Rectangle {
	return image.Rect(0, 0, m.Width*o.TileWidth, m.Height*o.TileHeight)
}

func (o Orthogonal) VisibleTiles(m *tilemap.Map, view image.Rectangle) (c0, r0, c1, r1 int) {
	c0 = max(0, floorDiv(view.Min.X, o.TileWidth))
	r0 = max(0, floorDiv(view.Min.Y, o.TileHeight))
	c1 = min(m.Width-1, floorDiv(view.Max.X-1, o.TileWidth))
	r1 = min(m.Height-1, floorDiv(view.Max.Y-1, o.TileHeight))
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
