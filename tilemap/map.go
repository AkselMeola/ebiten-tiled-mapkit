// Package tilemap is the format-agnostic map model. It must not import Ebiten or Tiled.
package tilemap

import (
	"image"
	"iter"
	"math"
	"time"
)

// TileID indexes Map.Tiles; 0 is an empty cell.
type TileID uint32

// ImageRef is a slash-separated path within the file system the map was loaded from.
type ImageRef string

type Orientation uint8

const (
	OrientationUnknown Orientation = iota
	OrientationOrthogonal
	OrientationIsometric
	OrientationStaggered
	OrientationHexagonal
)

type Flip uint8

const (
	FlipH Flip = 1 << iota
	FlipV
	FlipD
)

type Cell struct {
	TileID TileID
	Flip   Flip
}

func (c Cell) Empty() bool { return c.TileID == 0 }

type TileDef struct {
	Image     ImageRef
	Src       image.Rectangle
	Offset    image.Point
	Class     string
	Animation []Frame
	Props     Properties
	Colliders []Shape

	animTotal time.Duration
}

type Frame struct {
	TileID   TileID
	Duration time.Duration
}

func (d *TileDef) SetAnimation(frames []Frame) {
	d.Animation = frames
	d.animTotal = sumFrames(frames)
}

func (d *TileDef) AnimationDuration() time.Duration {
	if d.animTotal > 0 {
		return d.animTotal
	}
	return sumFrames(d.Animation)
}

// FrameAt returns the frame to show at clock t, or 0 for static tiles.
func (d *TileDef) FrameAt(t time.Duration) TileID {
	if len(d.Animation) == 0 {
		return 0
	}
	total := d.AnimationDuration()
	if total <= 0 {
		return d.Animation[0].TileID
	}

	t %= total
	if t < 0 {
		t += total
	}
	for _, f := range d.Animation {
		if t < f.Duration {
			return f.TileID
		}
		t -= f.Duration
	}
	return d.Animation[len(d.Animation)-1].TileID
}

// FrameAtOnce plays the animation once and holds the last frame; done reports it has finished.
// Static tiles return 0, true.
func (d *TileDef) FrameAtOnce(t time.Duration) (id TileID, done bool) {
	if len(d.Animation) == 0 {
		return 0, true
	}
	if t < 0 {
		return d.Animation[0].TileID, false
	}
	if t >= d.AnimationDuration() {
		return d.Animation[len(d.Animation)-1].TileID, true
	}
	return d.FrameAt(t), false
}

func sumFrames(frames []Frame) time.Duration {
	var total time.Duration
	for _, f := range frames {
		total += f.Duration
	}
	return total
}

type Map struct {
	Width, Height         int
	TileWidth, TileHeight int
	Orientation           Orientation
	Tiles                 []TileDef
	Layers                []Layer
	Props                 Properties
}

// TileDef returns nil for empty or unknown ids.
func (m *Map) TileDef(id TileID) *TileDef {
	if id == 0 || int(id) >= len(m.Tiles) {
		return nil
	}
	return &m.Tiles[id]
}

// InBounds reports whether p is on the map.
func (m *Map) InBounds(p TilePos) bool {
	return p.Col >= 0 && p.Row >= 0 && p.Col < m.Width && p.Row < m.Height
}

// Layer finds a layer by name, depth-first through groups.
func (m *Map) Layer(name string) Layer {
	return findLayer(m.Layers, name)
}

// AllLayers yields every layer depth-first, each group before its children.
func (m *Map) AllLayers() iter.Seq[Layer] {
	return func(yield func(Layer) bool) {
		walkLayers(m.Layers, yield)
	}
}

func walkLayers(layers []Layer, yield func(Layer) bool) bool {
	for _, l := range layers {
		if !yield(l) {
			return false
		}
		if g, ok := l.(*GroupLayer); ok && !walkLayers(g.Layers, yield) {
			return false
		}
	}
	return true
}

// TileLayers yields every tile layer, groups included, in draw order.
func (m *Map) TileLayers() iter.Seq[*TileLayer] {
	return func(yield func(*TileLayer) bool) {
		for l := range m.AllLayers() {
			if l, ok := l.(*TileLayer); ok && !yield(l) {
				return
			}
		}
	}
}

// Objects yields every object of every object layer, in layer order. Changes through the pointer
// stick, such as hiding a spawned object.
func (m *Map) Objects() iter.Seq[*MapObject] {
	return func(yield func(*MapObject) bool) {
		for l := range m.AllLayers() {
			if l, ok := l.(*ObjectLayer); ok {
				for i := range l.Objects {
					if !yield(&l.Objects[i]) {
						return
					}
				}
			}
		}
	}
}

// ObjectsAt yields the visible objects covering pos, in layer order. An object covers a tile when
// its bounds contain the tile's center; a point covers the tile it is on.
func (m *Map) ObjectsAt(pos TilePos) iter.Seq[*MapObject] {
	center := Vector{(float64(pos.Col) + 0.5) * float64(m.TileWidth), (float64(pos.Row) + 0.5) * float64(m.TileHeight)}
	return func(yield func(*MapObject) bool) {
		for o := range m.Objects() {
			if !o.Visible {
				continue
			}
			lo, hi := o.Bounds()
			covers := center.X >= lo.X && center.X < hi.X && center.Y >= lo.Y && center.Y < hi.Y
			if lo.X == hi.X || lo.Y == hi.Y {
				covers = m.WorldToTile(o.Center()) == pos
			}
			if covers && !yield(o) {
				return
			}
		}
	}
}

// WorldToTile is the tile at world position v, such as an object's or an entity's. For the tile an
// object stands on, use its center: m.WorldToTile(o.Center()).
func (m *Map) WorldToTile(v Vector) TilePos {
	return TilePos{
		Col: int(math.Floor(v.X / float64(m.TileWidth))),
		Row: int(math.Floor(v.Y / float64(m.TileHeight))),
	}
}

// TileToWorld is the world position of p's top-left corner.
func (m *Map) TileToWorld(p TilePos) Vector {
	return Vector{float64(p.Col * m.TileWidth), float64(p.Row * m.TileHeight)}
}

func findLayer(layers []Layer, name string) Layer {
	for _, l := range layers {
		if l.Base().Name == name {
			return l
		}
		if g, ok := l.(*GroupLayer); ok {
			if found := findLayer(g.Layers, name); found != nil {
				return found
			}
		}
	}
	return nil
}
