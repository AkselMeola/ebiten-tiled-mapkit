package render

import (
	"cmp"
	"errors"
	"fmt"
	"image"
	"math"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

type Camera struct {
	X, Y float64
}

// LayerHook runs after every layer, visible or not, so the game can draw its entities in between.
type LayerHook func(dst *ebiten.Image, l tilemap.Layer, cam Camera)

// MapRenderer draws one map. Create one per map: images are shared through the ImageSource,
// so a new renderer only cuts tiles from images already loaded.
type MapRenderer struct {
	AfterLayer LayerHook

	m        *tilemap.Map
	proj     Projection
	source   ImageSource
	clock    time.Duration
	tiles    []*ebiten.Image
	parents  []*ebiten.Image
	overdraw image.Rectangle
	order    []drawItem
	vertices []ebiten.Vertex
	indices  []uint32
}

// NewMapRenderer loads every image m uses and returns all load errors.
func NewMapRenderer(source ImageSource, m *tilemap.Map) (*MapRenderer, error) {
	if m.Orientation != tilemap.OrientationOrthogonal && m.Orientation != tilemap.OrientationUnknown {
		return nil, fmt.Errorf("render: only orthogonal maps are supported, got orientation %d", m.Orientation)
	}
	r := &MapRenderer{
		m:        m,
		proj:     NewOrthogonal(m),
		source:   source,
		tiles:    make([]*ebiten.Image, len(m.Tiles)),
		parents:  make([]*ebiten.Image, len(m.Tiles)),
		overdraw: overdraw(m),
	}
	var errs []error
	for id := 1; id < len(m.Tiles); id++ {
		if err := r.loadTile(tilemap.TileID(id)); err != nil {
			errs = append(errs, err)
		}
	}
	for l := range m.AllLayers() {
		if l, ok := l.(*tilemap.ImageLayer); ok && l.Image != "" {
			if _, err := source.Image(l.Image); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return r, nil
}

// Update advances the animation clock by one tick. Call it from ebiten's Update.
func (r *MapRenderer) Update() {
	r.clock += Tick()
}

// Clock is the animation time, the sum of all Update ticks.
func (r *MapRenderer) Clock() time.Duration {
	return r.clock
}

// Projection converts between tile and screen positions for this map.
func (r *MapRenderer) Projection() Projection {
	return r.proj
}

// Tick is the fixed duration of one ebiten update.
func Tick() time.Duration {
	return time.Second / time.Duration(ebiten.TPS())
}

// TileAtScreen is the tile drawn at screen pixel x, y with camera cam, such as the one under the
// mouse: x, y := ebiten.CursorPosition().
func (r *MapRenderer) TileAtScreen(x, y int, cam Camera) tilemap.TilePos {
	return r.proj.ScreenToTile(float64(x)+math.Round(cam.X), float64(y)+math.Round(cam.Y))
}

// Draw draws the map with the camera rounded to whole pixels; AfterLayer receives the rounded camera.
func (r *MapRenderer) Draw(dst *ebiten.Image, cam Camera) {
	cam = Camera{X: math.Round(cam.X), Y: math.Round(cam.Y)}
	r.drawLayers(dst, r.m.Layers, cam, 0, 0, 1)
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

	c0, r0, c1, r1 := r.proj.VisibleTiles(r.m, view)
	c1, r1 = min(c1, l.W-1), min(r1, l.H-1)
	th := float64(r.m.TileHeight)
	t := r.clock

	// Tiles are batched into one DrawTriangles call per run of tiles sharing a tileset image.
	var batch *ebiten.Image
	for row := r0; row <= r1; row++ {
		for col := c0; col <= c1; col++ {
			cell := l.Cells[row*l.W+col]
			if cell.Empty() {
				continue
			}
			id, img := r.frame(cell.TileID, t)
			if img == nil {
				continue
			}
			if tex := r.parents[id]; tex != batch {
				r.flush(dst, batch)
				batch = tex
			}
			def := &r.m.Tiles[id]
			src := img.Bounds()
			_, h := flippedSize(cell.Flip, src)
			x, y := r.proj.TileToScreen(tilemap.TilePos{Col: col, Row: row})
			r.appendQuad(
				x+float64(def.Offset.X)+ox-cam.X,
				y+th-h+float64(def.Offset.Y)+oy-cam.Y,
				src, cell.Flip, float32(alpha),
			)
		}
	}
	r.flush(dst, batch)
}

type drawItem struct {
	index int
	sortY float64
}

// drawOrder lists the visible tile objects of l in the layer's draw order, reusing buf.
func drawOrder(l *tilemap.ObjectLayer, buf []drawItem) []drawItem {
	buf = buf[:0]
	for i := range l.Objects {
		o := &l.Objects[i]
		if !o.Visible || !o.IsTile() {
			continue
		}
		item := drawItem{index: i}
		if l.DrawOrder == tilemap.DrawOrderTopDown {
			item.sortY = o.SortY()
		}
		buf = append(buf, item)
	}
	if l.DrawOrder == tilemap.DrawOrderTopDown {
		slices.SortStableFunc(buf, func(a, b drawItem) int { return cmp.Compare(a.sortY, b.sortY) })
	}
	return buf
}

// appendQuad adds the tile at src with its top-left corner at x, y, flipped as DrawTileOnScreen does.
func (r *MapRenderer) appendQuad(x, y float64, src image.Rectangle, flip tilemap.Flip, alpha float32) {
	var g ebiten.GeoM
	applyFlips(&g, flip, src)
	// Whole pixels: unlike DrawImage, DrawTriangles doesn't snap vertices, so edges would sample neighboring tiles.
	g.Translate(math.Round(x), math.Round(y))

	base := uint32(len(r.vertices))
	w, h := float64(src.Dx()), float64(src.Dy())
	for _, corner := range [4][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		dx, dy := g.Apply(corner[0], corner[1])
		r.vertices = append(r.vertices, ebiten.Vertex{
			DstX:   float32(dx),
			DstY:   float32(dy),
			SrcX:   float32(float64(src.Min.X) + corner[0]),
			SrcY:   float32(float64(src.Min.Y) + corner[1]),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha,
		})
	}
	r.indices = append(r.indices, base, base+1, base+2, base+1, base+3, base+2)
}

func (r *MapRenderer) flush(dst, tex *ebiten.Image) {
	if len(r.indices) > 0 {
		dst.DrawTriangles32(r.vertices, r.indices, tex, &ebiten.DrawTrianglesOptions{})
	}
	r.vertices, r.indices = r.vertices[:0], r.indices[:0]
}

func (r *MapRenderer) drawObjectLayer(dst *ebiten.Image, l *tilemap.ObjectLayer, cam Camera, ox, oy, alpha float64) {
	r.order = drawOrder(l, r.order)
	for _, item := range r.order {
		o := &l.Objects[item.index]
		id, img := r.frame(o.Cell.TileID, r.clock)
		if img == nil {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		w, h := flippedSize(o.Cell.Flip, img.Bounds())
		if o.Size.X > 0 && o.Size.Y > 0 && w > 0 && h > 0 {
			op.GeoM.Scale(o.Size.X/w, o.Size.Y/h)
		}
		op.GeoM.Rotate(o.Rotation * math.Pi / 180)
		op.GeoM.Translate(o.Pos.X+ox-cam.X, o.Pos.Y+oy-cam.Y)
		op.ColorScale.ScaleAlpha(float32(alpha * o.Opacity))
		r.drawTile(dst, img, &r.m.Tiles[id], o.Cell.Flip, op)
	}
}

// DrawTileInWorld draws cell's tile at animation time t with its top-left corner at world position
// pos, seen from cam, for game entities. op may be nil; its GeoM adjusts the tile around its
// top-left corner before placing, and its colors and blending apply as given.
func (r *MapRenderer) DrawTileInWorld(dst *ebiten.Image, cell tilemap.Cell, pos tilemap.Vector, cam Camera, t time.Duration, op *ebiten.DrawImageOptions) {
	placed := inWorld(op, pos, cam)
	r.DrawTileOnScreen(dst, cell, t, &placed)
}

// inWorld copies op, so the caller's stays unchanged, and adds the move to the screen. The camera
// is rounded as in Draw, so entities line up with the map.
func inWorld(op *ebiten.DrawImageOptions, pos tilemap.Vector, cam Camera) ebiten.DrawImageOptions {
	var placed ebiten.DrawImageOptions
	if op != nil {
		placed = *op
	}
	placed.GeoM.Translate(pos.X-math.Round(cam.X), pos.Y-math.Round(cam.Y))
	return placed
}

// DrawTileOnScreen draws cell's tile at animation time t, with its top-left corner at the origin
// of op.GeoM in screen pixels, for HUDs and custom transforms. Flips and the tile offset are
// applied before op; op may be nil.
func (r *MapRenderer) DrawTileOnScreen(dst *ebiten.Image, cell tilemap.Cell, t time.Duration, op *ebiten.DrawImageOptions) {
	id, img := r.frame(cell.TileID, t)
	if img == nil {
		return
	}
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	r.drawTile(dst, img, &r.m.Tiles[id], cell.Flip, op)
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
	img, err := r.source.Image(l.Image)
	if err != nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox-cam.X, oy-cam.Y)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(img, op)
}

// frame resolves the animation at t and returns the tile to draw with its image.
func (r *MapRenderer) frame(id tilemap.TileID, t time.Duration) (tilemap.TileID, *ebiten.Image) {
	if id == 0 || int(id) >= len(r.tiles) {
		return id, nil
	}
	if def := &r.m.Tiles[id]; def.Animation != nil {
		if f := def.FrameAt(t); f != 0 && int(f) < len(r.tiles) {
			id = f
		}
	}
	return id, r.tiles[id]
}

func (r *MapRenderer) loadTile(id tilemap.TileID) error {
	def := &r.m.Tiles[id]
	if def.Image == "" {
		return nil
	}
	tex, err := r.source.Image(def.Image)
	if err != nil {
		return err
	}
	r.parents[id] = tex
	if def.Src.Empty() {
		r.tiles[id] = tex
	} else {
		r.tiles[id] = tex.SubImage(def.Src).(*ebiten.Image)
	}
	return nil
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
