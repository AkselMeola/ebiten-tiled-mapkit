package render

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// Sprite is a game entity drawn among a layer's tiles and tile objects in depth order, so it can
// walk behind a tree. A tile stands at its center; a sprite at the same depth goes on top, and on
// top of every object in a DrawOrderIndex layer.
type Sprite interface {
	// Foot is the bottom-centre point the sprite stands on, in world pixels. The renderer sorts
	// by it and passes its screen position to Draw.
	Foot() tilemap.Vector
	// Draw draws the sprite with its foot at screen position x, y.
	Draw(dst *ebiten.Image, x, y float64)
}

// SpritesByLayer is the sprites to draw in each layer, by layer name, for MapRenderer.Draw.
type SpritesByLayer map[string][]Sprite

// TileSprite is a Sprite that draws one tile standing on Pos. Build one per frame from the game's
// entity. ColorScale tints it and GeoM squashes, scales or turns it; for anything else, embed it
// in your own type, override Draw and draw Image yourself.
type TileSprite struct {
	Renderer *MapRenderer
	// Pos is the bottom-centre point the tile stands on, in world pixels: its Foot.
	Pos tilemap.Vector
	// Cell is the tile to draw as is, usually Cell{TileID: animator.Frame()}. Its animation isn't
	// resolved again: an animation's frame is often the animated tile itself.
	Cell tilemap.Cell
	// ColorScale tints the tile, e.g. ColorScale.Scale(0.6, 1, 0.6, 1); the zero value doesn't.
	ColorScale ebiten.ColorScale
	// GeoM transforms the tile around its foot before it is placed, e.g. GeoM.Scale(1.2, 0.8)
	// squashes it onto the ground; the zero value leaves it as is.
	GeoM ebiten.GeoM
}

func (s TileSprite) Foot() tilemap.Vector { return s.Pos }

// Draw draws the tile bottom-centred on x, y, flipped and offset as in the map, then transformed
// by GeoM around x, y.
func (s TileSprite) Draw(dst *ebiten.Image, x, y float64) {
	img := s.Image()
	if img == nil {
		return
	}
	w, h := flippedSize(s.Cell.Flip, img.Bounds())
	op := &ebiten.DrawImageOptions{GeoM: s.place(w, h, x, y)}
	op.ColorScale = s.ColorScale
	s.Renderer.drawTile(dst, img, &s.Renderer.m.Tiles[s.Cell.TileID], s.Cell.Flip, op)
}

// place moves a w by h tile's foot to the origin, applies GeoM there, then moves it to x, y.
func (s TileSprite) place(w, h, x, y float64) ebiten.GeoM {
	var g ebiten.GeoM
	g.Translate(-w/2, -h)
	g.Concat(s.GeoM)
	g.Translate(x, y)
	return g
}

// Image is Cell's own tile image, without its animation resolved again; nil for an empty or
// unknown tile. Flips and the tile offset aren't applied: a Draw override that wants them calls
// TileSprite.Draw, which applies GeoM, instead.
func (s TileSprite) Image() *ebiten.Image {
	id := s.Cell.TileID
	if id == 0 || int(id) >= len(s.Renderer.tiles) {
		return nil
	}
	return s.Renderer.tiles[id]
}
