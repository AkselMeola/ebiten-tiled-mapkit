# ebiten-tiled-mapkit

Load [Tiled](https://www.mapeditor.org/) maps and draw them with [Ebitengine](https://ebitengine.org/).

```
.tmx file ──► tiled.Load ──► tilemap.Map ──► render.MapRenderer.Draw ──► screen
              (translate)     (plain data)    (draw)
```

- **Plain data model.** A map loads into `tilemap.Map`, plain Go data with no Ebiten or Tiled
  types. Your game logic reads it directly (collision, spawns, properties), and other formats can
  produce the same model.
- **Renderer.** Orthogonal projection, culling, cached sub-images, tile animations, flips, layer
  opacity and offsets, group layers, image layers and tile objects.
- **Entity-friendly.** Draw your own entities with the map's tiles and animations, between any two
  layers.
- **Any `fs.FS`.** Works with `embed.FS` (single binary, WebAssembly) and `os.DirFS`.

## Install

```bash
go get github.com/AkselMeola/ebiten-tiled-mapkit
```

Requires Go 1.25+ and Ebitengine v2.

## Quick start

```go
package main

import (
	"embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/render"
	"github.com/AkselMeola/ebiten-tiled-mapkit/tiled"
	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

//go:embed maps/*.tmx maps/*.tsx images/*.png
var assets embed.FS

type game struct {
	m        *tilemap.Map
	anim     *render.Animator
	renderer *render.MapRenderer
	cam      render.Camera
}

func main() {
	m, err := tiled.Load(assets, "maps/level1.tmx")
	if err != nil {
		log.Fatal(err)
	}

	anim := &render.Animator{}
	r := render.NewMapRenderer(render.NewOrthogonal(m), render.NewFSTextures(assets), anim)
	if err := r.Preload(m); err != nil { // optional: fail early on missing images
		log.Fatal(err)
	}

	if err := ebiten.RunGame(&game{m: m, anim: anim, renderer: r}); err != nil {
		log.Fatal(err)
	}
}

func (g *game) Update() error {
	g.anim.Update(render.Tick())
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.renderer.Draw(screen, g.m, g.cam)
}

func (g *game) Layout(w, h int) (int, int) { return w, h }
```

Image and tileset paths in the map are resolved relative to the map file, inside the same `fs.FS`.
Embed everything the map references.

## Example

[`examples/coins`](examples/coins) is a small game: walk around a Tiled map, blocked by walls, and
collect animated coins.

```bash
go run github.com/AkselMeola/ebiten-tiled-mapkit/examples/coins@latest
# or, in a clone:
go run ./examples/coins
```

## Using map data in your game

### Tiles and collision

Each tile layer is a grid of `Cell`s. A cell's `TileID` indexes `Map.Tiles`, whose `TileDef` carries
the tile's class (set as *Class* or *Type* on a tile in Tiled) and custom properties.

```go
ground := m.Layer("ground").(*tilemap.TileLayer)

func blocked(col, row int) bool {
	def := m.Tile(ground.At(col, row).Tile) // nil for empty cells and outside the layer
	return def == nil || def.Class == "Wall" || def.Props.Bool("solid")
}
```

`Projection.ScreenToTile` and `TileToScreen` convert between pixels and cells.

### Objects

Object layers hold `MapObject`s: spawn points, triggers, items. They are descriptors; your game
creates its own entities from them.

```go
for i := range objects.Objects {
	o := &objects.Objects[i]
	switch o.Type {
	case "Player":
		player = newPlayer(o.X, o.Y)
		o.Visible = false // the game draws the player from now on
	case "Enemy":
		spawnEnemy(o.X, o.Y, o.Props.Int("hp"))
	}
}
```

`X, Y` is always the object's **top-left corner**, and `Rotation` turns around it. Tiled's own
anchor for tile objects (bottom-left by default) is converted on load. `Bounds()` gives the rotated
bounding box and `SortY()` its bottom edge for depth sorting.

Tile objects are drawn by the renderer. Hide one with `Visible = false`, for example when a
coin is collected.

### Properties

Custom properties keep their Tiled types:

| Tiled type | Go value | Getter |
|---|---|---|
| string | `string` | `Props.String` |
| int | `int` | `Props.Int` |
| float | `float64` | `Props.Float` |
| bool | `bool` | `Props.Bool` |
| color | `color.RGBA` | `Props.Color` |
| file | `tilemap.FilePath` | `Props.String` |
| object | `tilemap.ObjectRef` | `Props.Int` |
| class | `tilemap.Properties` | `Props.Class` |

Maps, layers, tiles and objects all have `Props`.

## Drawing your own entities

Entities can use the map's tiles and animations, drawn between map layers:

```go
r.AfterLayer = func(dst *ebiten.Image, l tilemap.Layer, cam render.Camera) {
	if l.Base().Name != "objects" {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Rotate(player.angle)
	op.GeoM.Translate(player.x-cam.X, player.y-cam.Y)
	r.DrawTile(dst, m, tilemap.Cell{Tile: player.tile, Flip: player.flip}, anim.Clock-player.animStart, op)
}
```

- `AfterLayer` runs after every layer, visible or not, in draw order.
- `DrawTile` applies the cell's flips and the tile offset, then your `op`. The tile's top-left
  corner is at the origin of `op.GeoM`.
- The time argument picks the animation frame. `anim.Clock` keeps the tile in sync with the map,
  and `anim.Clock - start` plays the animation from its first frame since `start`. Restarting,
  pausing or changing speed is just changing that number.

For play-once animations (a door opening, a box landing), use `TileDef.FrameAtOnce(t)`, which holds
the last frame and reports when it's done.

To switch animation by state or direction, give each animated tile properties in Tiled
(`state=walk`, `dir=left`) and build a lookup from `Map.Tiles` after loading.

## Camera and zoom

`Camera{X, Y}` is the world position of the screen's top-left corner. For zoom, return a smaller
logical size from `Layout` and let Ebiten scale, or draw to an offscreen image and scale that.

## Packages

| Package | Purpose |
|---|---|
| `tilemap` | Data model: `Map`, `TileDef`, layers, `MapObject`, `Properties`. No Ebiten or Tiled imports. |
| `tiled` | `Load(fsys, path)`: Tiled `.tmx` → `tilemap.Map`. |
| `render` | `MapRenderer`, `Projection` (`Orthogonal`), `FSTextures`, `Animator`. |

## Tiled support

| Supported | Not yet |
|---|---|
| Orthogonal maps | Isometric, staggered, hexagonal rendering (they load, but only an orthogonal projection exists) |
| CSV, XML and base64 layer data, zlib and gzip | zstd, infinite maps, `.tmj` JSON maps |
| Embedded and external tilesets; single-image and image-collection tilesets | Tileset transparent color (`trans`) |
| Tile animations, flips and rotations, tile offsets, per-tile collision shapes | Wang sets and terrains (ignored) |
| Tile, rectangle, ellipse, point, polygon and polyline objects; object templates | Text objects (loaded as rectangles) |
| Tile, object, image and group layers with visibility, opacity and offsets | Parallax, tint, image layer repeat |
| Custom properties, including class properties | Worlds (`.world`) |

## Development

```bash
go test ./...
go run ./examples/coins
```

Architecture, decisions and roadmap: [`docs/DESIGN.md`](docs/DESIGN.md).

## License

[MIT](LICENSE)
