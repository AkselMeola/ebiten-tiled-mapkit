package render

import (
	"errors"
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

type Camera struct {
	X, Y float64
}

// LayerHook runs after every layer, visible or not, so the game can draw its entities in between.
type LayerHook func(dst *ebiten.Image, l tilemap.Layer, cam Camera)

type MapRenderer struct {
	Proj       Projection
	Textures   TextureProvider
	Anim       *Animator
	AfterLayer LayerHook

	m        *tilemap.Map
	tiles    []*ebiten.Image
	tried    []bool
	overdraw image.Rectangle
	err      error
}

func NewMapRenderer(proj Projection, textures TextureProvider, anim *Animator) *MapRenderer {
	return &MapRenderer{Proj: proj, Textures: textures, Anim: anim}
}

// Err returns the texture errors met while drawing.
func (r *MapRenderer) Err() error {
	return r.err
}

// Preload loads every tile texture of m up front.
func (r *MapRenderer) Preload(m *tilemap.Map) error {
	r.prepare(m)
	var errs []error
	for id := 1; id < len(m.Tiles); id++ {
		if _, err := r.loadTile(tilemap.TileID(id)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *MapRenderer) Draw(dst *ebiten.Image, m *tilemap.Map, cam Camera) {
	r.prepare(m)
	r.drawLayers(dst, m.Layers, cam, 0, 0, 1)
}

func (r *MapRenderer) drawLayers(dst *ebiten.Image, layers []tilemap.Layer, cam Camera, ox, oy, alpha float64) {
	for _, l := range layers {
		b := l.Base()
		if b.Visible {
			lx, ly, la := ox+b.OffsetX, oy+b.OffsetY, alpha*b.Opacity
			switch l := l.(type) {
			case *tilemap.TileLayer:
				r.drawTileLayer(dst, l, cam, lx, ly, la)
			case *tilemap.ObjectLayer:
				r.drawObjectLayer(dst, l, cam, lx, ly, la)
			case *tilemap.ImageLayer:
				r.drawImageLayer(dst, l, cam, lx, ly, la)
			case *tilemap.GroupLayer:
				r.drawLayers(dst, l.Layers, cam, lx, ly, la)
			}
		}
		if r.AfterLayer != nil {
			r.AfterLayer(dst, l, cam)
		}
	}
}

func (r *MapRenderer) drawTileLayer(dst *ebiten.Image, l *tilemap.TileLayer, cam Camera, ox, oy, alpha float64) {
	b := dst.Bounds()
	minX, minY := int(math.Floor(cam.X-ox)), int(math.Floor(cam.Y-oy))
	view := image.Rect(
		minX-r.overdraw.Max.X, minY-r.overdraw.Max.Y,
		minX+b.Dx()+r.overdraw.Min.X, minY+b.Dy()+r.overdraw.Min.Y,
	)

	c0, r0, c1, r1 := r.Proj.VisibleTiles(r.m, view)
	c1, r1 = min(c1, l.W-1), min(r1, l.H-1)
	th := float64(r.m.TileHeight)

	for row := r0; row <= r1; row++ {
		for col := c0; col <= c1; col++ {
			cell := l.Cells[row*l.W+col]
			if cell.Empty() {
				continue
			}
			id, img := r.frame(cell.Tile, r.clock())
			if img == nil {
				continue
			}
			def := &r.m.Tiles[id]

			op := &ebiten.DrawImageOptions{}
			_, h := applyFlips(&op.GeoM, cell.Flip, img.Bounds())
			x, y := r.Proj.TileToScreen(col, row)
			op.GeoM.Translate(
				x+float64(def.Offset.X)+ox-cam.X,
				y+th-h+float64(def.Offset.Y)+oy-cam.Y,
			)
			op.ColorScale.ScaleAlpha(float32(alpha))
			dst.DrawImage(img, op)
		}
	}
}

func (r *MapRenderer) drawObjectLayer(dst *ebiten.Image, l *tilemap.ObjectLayer, cam Camera, ox, oy, alpha float64) {
	for i := range l.Objects {
		o := &l.Objects[i]
		if !o.Visible || !o.IsTile() {
			continue
		}
		id, img := r.frame(o.Tile.Tile, r.clock())
		if img == nil {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		w, h := flippedSize(o.Tile.Flip, img.Bounds())
		if o.W > 0 && o.H > 0 && w > 0 && h > 0 {
			op.GeoM.Scale(o.W/w, o.H/h)
		}
		op.GeoM.Rotate(o.Rotation * math.Pi / 180)
		op.GeoM.Translate(o.X+ox-cam.X, o.Y+oy-cam.Y)
		op.ColorScale.ScaleAlpha(float32(alpha * o.Opacity))
		r.drawTile(dst, img, &r.m.Tiles[id], o.Tile.Flip, op)
	}
}

// DrawTile draws cell's tile at animation time t, with its top-left corner at the origin of op.GeoM.
// Flips and the tile offset are applied before op; op may be nil.
func (r *MapRenderer) DrawTile(dst *ebiten.Image, m *tilemap.Map, cell tilemap.Cell, t time.Duration, op *ebiten.DrawImageOptions) {
	r.prepare(m)
	id, img := r.frame(cell.Tile, t)
	if img == nil {
		return
	}
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	r.drawTile(dst, img, &m.Tiles[id], cell.Flip, op)
}

func (r *MapRenderer) drawTile(dst, img *ebiten.Image, def *tilemap.TileDef, flip tilemap.Flip, op *ebiten.DrawImageOptions) {
	local := *op
	local.GeoM.Reset()
	applyFlips(&local.GeoM, flip, img.Bounds())
	local.GeoM.Translate(float64(def.Offset.X), float64(def.Offset.Y))
	local.GeoM.Concat(op.GeoM)
	dst.DrawImage(img, &local)
}

func (r *MapRenderer) drawImageLayer(dst *ebiten.Image, l *tilemap.ImageLayer, cam Camera, ox, oy, alpha float64) {
	if l.Image == "" {
		return
	}
	img, err := r.Textures.Texture(l.Image)
	if err != nil {
		r.setErr(err)
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox-cam.X, oy-cam.Y)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(img, op)
}

func (r *MapRenderer) clock() time.Duration {
	if r.Anim == nil {
		return 0
	}
	return r.Anim.Clock
}

// frame resolves the animation at t and returns the tile to draw with its cached image.
func (r *MapRenderer) frame(id tilemap.TileID, t time.Duration) (tilemap.TileID, *ebiten.Image) {
	if id == 0 || int(id) >= len(r.m.Tiles) {
		return id, nil
	}
	if def := &r.m.Tiles[id]; def.Animation != nil {
		if f := def.FrameAt(t); f != 0 && int(f) < len(r.m.Tiles) {
			id = f
		}
	}
	img, err := r.loadTile(id)
	if err != nil {
		r.setErr(err)
	}
	return id, img
}

func (r *MapRenderer) loadTile(id tilemap.TileID) (*ebiten.Image, error) {
	if r.tried[id] {
		return r.tiles[id], nil
	}
	r.tried[id] = true

	def := &r.m.Tiles[id]
	if def.Image == "" {
		return nil, nil
	}
	tex, err := r.Textures.Texture(def.Image)
	if err != nil {
		return nil, err
	}
	if def.Src.Empty() {
		r.tiles[id] = tex
	} else {
		r.tiles[id] = tex.SubImage(def.Src).(*ebiten.Image)
	}
	return r.tiles[id], nil
}

func (r *MapRenderer) setErr(err error) {
	if r.err == nil {
		r.err = err
	}
}

// prepare resets the per-map caches when a different map is drawn.
func (r *MapRenderer) prepare(m *tilemap.Map) {
	if r.m == m && len(r.tiles) == len(m.Tiles) {
		return
	}
	r.m = m
	r.tiles = make([]*ebiten.Image, len(m.Tiles))
	r.tried = make([]bool, len(m.Tiles))
	r.err = nil
	r.overdraw = overdraw(m)
}

// overdraw is how far tiles reach outside their cell: Min is left/top, Max is right/bottom.
func overdraw(m *tilemap.Map) image.Rectangle {
	var o image.Rectangle
	for i := 1; i < len(m.Tiles); i++ {
		def := &m.Tiles[i]
		s := max(def.Src.Dx(), def.Src.Dy())
		o.Min.X = max(o.Min.X, -def.Offset.X)
		o.Max.X = max(o.Max.X, def.Offset.X+s-m.TileWidth)
		o.Min.Y = max(o.Min.Y, s-m.TileHeight-def.Offset.Y)
		o.Max.Y = max(o.Max.Y, def.Offset.Y)
	}
	return o
}

func flippedSize(f tilemap.Flip, bounds image.Rectangle) (w, h float64) {
	w, h = float64(bounds.Dx()), float64(bounds.Dy())
	if f&tilemap.FlipD != 0 {
		return h, w
	}
	return w, h
}

// applyFlips flips img in place within its bounds and returns the drawn size. Diagonal goes first.
func applyFlips(g *ebiten.GeoM, f tilemap.Flip, bounds image.Rectangle) (w, h float64) {
	w, h = float64(bounds.Dx()), float64(bounds.Dy())
	if f&tilemap.FlipD != 0 {
		g.Rotate(math.Pi / 2)
		g.Scale(-1, 1)
		w, h = h, w
	}
	if f&tilemap.FlipH != 0 {
		g.Scale(-1, 1)
		g.Translate(w, 0)
	}
	if f&tilemap.FlipV != 0 {
		g.Scale(1, -1)
		g.Translate(0, h)
	}
	return w, h
}
