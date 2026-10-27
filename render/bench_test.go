package render

import (
	"image"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

type memImages struct{ img *ebiten.Image }

func (t memImages) Image(tilemap.ImageRef) (*ebiten.Image, error) { return t.img, nil }

// benchMap is 200x200 tiles of 16 px with three full tile layers and some animated tiles.
func benchMap() *tilemap.Map {
	const size, tile, cols = 200, 16, 16
	m := &tilemap.Map{Width: size, Height: size, TileWidth: tile, TileHeight: tile, Tiles: make([]tilemap.TileDef, 1, cols*cols+1)}
	for i := range cols * cols {
		x, y := i%cols*tile, i/cols*tile
		m.Tiles = append(m.Tiles, tilemap.TileDef{Image: "t", Src: image.Rect(x, y, x+tile, y+tile)})
	}
	m.Tiles[1].SetAnimation([]tilemap.Frame{{Tile: 1, Duration: 100 * time.Millisecond}, {Tile: 2, Duration: 100 * time.Millisecond}})

	for layer := range 3 {
		cells := make([]tilemap.Cell, size*size)
		for i := range cells {
			cells[i] = tilemap.Cell{TileID: tilemap.TileID(1 + (i*7+layer)%(cols*cols))}
		}
		m.Layers = append(m.Layers, &tilemap.TileLayer{LayerBase: tilemap.LayerBase{Visible: true, Opacity: 1}, W: size, H: size, Cells: cells})
	}
	return m
}

// BenchmarkDrawFullHD draws a 1920x1080 view of three 16 px tile layers, one frame per iteration.
// Frames are paced by the display, so ns/op is the frame time; draw-ns/op and draw-allocs/op
// measure Draw alone. Run with -bench; it needs a display.
func BenchmarkDrawFullHD(b *testing.B) {
	m := benchMap()
	var r *MapRenderer
	var dst *ebiten.Image
	inFrame(func() {
		var err error
		if r, err = NewMapRenderer(memImages{ebiten.NewImage(256, 256)}, m); err != nil {
			b.Fatal(err)
		}
		dst = ebiten.NewImage(1920, 1080)
	})
	cam := Camera{X: 500, Y: 500}

	var drawTime time.Duration
	var drawAllocs uint64
	var before, after runtime.MemStats
	for b.Loop() {
		inFrame(func() {
			r.Update()
			runtime.ReadMemStats(&before)
			start := time.Now()
			r.Draw(dst, cam)
			drawTime += time.Since(start)
			runtime.ReadMemStats(&after)
			drawAllocs += after.Mallocs - before.Mallocs
		})
	}
	b.ReportMetric(float64(drawTime.Nanoseconds())/float64(b.N), "draw-ns/op")
	b.ReportMetric(float64(drawAllocs)/float64(b.N), "draw-allocs/op")
}
