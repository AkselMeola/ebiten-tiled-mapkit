# ebiten-tiled-mapkit

Load [Tiled](https://www.mapeditor.org/) maps and draw them with [Ebitengine](https://ebitengine.org/).

> [!WARNING]
> Work in progress. The API may change between minor versions, and the library still needs battle
> testing in real games. It is not yet meant for wide use.

mapkit aims to be a generic tile map adapter and renderer: map formats are loaded through adapters
into one plain data model, which a renderer draws with Ebiten. Tiled is the first adapter. It is
meant to stay lightweight: map loading and drawing only, no game logic, physics or entity system.

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
	renderer *render.MapRenderer
	cam      render.Camera
}

func main() {
	cache := render.NewImageCache(assets) // for the whole game: maps and UI images

	m, err := tiled.Load(assets, "maps/level1.tmx")
	if err != nil {
		log.Fatal(err)
	}
	r, err := render.NewMapRenderer(cache, m) // loads the map's images, reports missing ones
	if err != nil {
		log.Fatal(err)
	}

	if err := ebiten.RunGame(&game{m: m, renderer: r}); err != nil {
		log.Fatal(err)
	}
}

func (g *game) Update() error {
	g.renderer.Update() // advances tile animations
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.renderer.Draw(screen, g.cam)
}

func (g *game) Layout(w, h int) (int, int) { return w, h }
```

Image and tileset paths in the map are resolved relative to the map file, inside the same `fs.FS`.
Embed everything the map references.

### Lifetimes

- **The image cache** (`ImageCache`) lives as long as the game. Create it at startup; UI images
  (`render.Images`) load through it too.
- **A map and its renderer** live as long as the level. A renderer draws one map; for another
  level, load the map and create a new renderer with the same cache. Images already loaded
  are reused, so that is cheap.

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

Each tile layer is a grid of `Cell`s. A cell's `TileID` indexes `Map.Tiles`; `m.TileDef(id)` returns
the `TileDef` with the tile's class (set as *Class* or *Type* on a tile in Tiled) and custom
properties. Positions on the grid are `tilemap.TilePos{Col, Row}`, comparable and usable as map keys.

```go
ground := m.Layer("ground").(*tilemap.TileLayer)

func blocked(pos tilemap.TilePos) bool {
	def := m.TileDef(ground.At(pos).TileID) // nil for empty cells and outside the layer
	return def == nil || def.Class == "Wall" || def.Props.Bool("solid")
}

if !blocked(playerTile.Add(tilemap.Right)) { ... } // Up, Down, Left, Right
```

To check every layer, for example to build a wall index once per level:

```go
walls := map[tilemap.TilePos]bool{}
for layer := range m.TileLayers() { // groups included, in draw order
	for pos, cell := range layer.All() { // non-empty cells, row by row
		if m.TileDef(cell.TileID).Class == "Wall" {
			walls[pos] = true
		}
	}
}
```

- `layer.Index(pos)` and `layer.Pos(i)` convert between positions and indexes in `layer.Cells`.
- `m.InBounds(pos)` checks a position against the map size.
- `Map.AllLayers()` yields every layer of any kind, for anything else.

Converting pixels to tiles:

```go
x, y := ebiten.CursorPosition()
hovered := renderer.TileAtScreen(x, y, cam) // the tile drawn under the mouse

standing := renderer.Projection().ScreenToTile(player.Pos.X, player.Pos.Y) // world pixels
```

### Objects

Object layers hold `MapObject`s: spawn points, triggers, items. They are descriptors; your game
creates its own entities from them.

```go
for o := range m.Objects() { // every object layer, groups included
	switch o.Type {
	case "Player":
		player = newPlayer(o.Pos)
		o.Visible = false // the game draws the player from now on
	case "Enemy":
		spawnEnemy(o.Pos, o.Props.Int("hp"))
	}
}
```

`Pos` is always the object's **top-left corner**, and `Rotation` turns around it. Tiled's own
anchor for tile objects (bottom-left by default) is converted on load. `Size` is the width and
height. `Center()` and `Foot()` (bottom center, where a character stands) take rotation into
account, `Bounds()` gives the rotated bounding box and `SortY()` its bottom edge for depth sorting.

Positions and sizes are `tilemap.Vector` (`struct{ X, Y float64 }`) with `Add`, `Sub`, `Scale` and
`Rotate`. Vectors compare with `==`, and a game's own vector type with the same fields converts
directly: `myVector(o.Pos)`.

Tile objects are drawn by the renderer. Hide one with `Visible = false`, for example when a
coin is collected. They are drawn in the layer's *Drawing Order* from Tiled: *Top Down* (the
default) sorts by `SortY()`, so lower objects overlap higher ones; *Manual* keeps the file order.

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
	op.GeoM.Translate(player.pos.X-cam.X, player.pos.Y-cam.Y)
	r.DrawTile(dst, tilemap.Cell{TileID: player.tile, Flip: player.flip}, r.Clock()-player.animStart, op)
}
```

- `AfterLayer` runs after every layer, visible or not, in draw order.
- `DrawTile` applies the cell's flips and the tile offset, then your `op`. The tile's top-left
  corner is at the origin of `op.GeoM`.
- The time argument picks the animation frame. `r.Clock()` keeps the tile in sync with the map,
  and `r.Clock() - start` plays the animation from its first frame since `start`. Restarting,
  pausing or changing speed is just changing that number.

For play-once animations (a door opening, a box landing), use `TileDef.FrameAtOnce(t)`, which holds
the last frame and reports when it's done.

To switch animation by state or direction, give each animated tile properties in Tiled
(`state=walk`, `dir=left`) and build a lookup from `Map.Tiles` after loading.

## Other images

`render.Images` holds named images outside the map, such as UI graphics. It needs no map: create
it at startup with the game's image cache, so images used by both are loaded once.

```go
cache := render.NewImageCache(assets)

ui := render.NewImages(cache).
	Add("button", "images/ui/button.png").
	Add("heart", "images/ui/heart.png").
	AddRegion("checkbox", "images/ui/sheet.png", image.Rect(0, 0, 16, 16)) // part of a sheet
if err := ui.Preload(); err != nil { // optional: fail early on missing images
	log.Fatal(err)
}

screen.DrawImage(ui.Get("heart"), op)
```

`Get` loads on first use and panics on an unknown name or a failed load, so call `Preload` at
startup to handle errors there.

## Camera and zoom

`Camera{X, Y}` is the world position of the screen's top-left corner. The renderer rounds it to
whole pixels, and passes the rounded camera to `AfterLayer`, so draw entities relative to that one.
For zoom, return a smaller logical size from `Layout` and let Ebiten scale, or draw to an offscreen
image and scale that.

## Packages

| Package | Purpose |
|---|---|
| `tilemap` | Data model: `Map`, `TileDef`, layers, `MapObject`, `Properties`. No Ebiten or Tiled imports. |
| `tiled` | `Load(fsys, path)`: Tiled `.tmx` → `tilemap.Map`. |
| `render` | `MapRenderer`, `Projection` (`Orthogonal`), `ImageCache`, `Images`. |

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
