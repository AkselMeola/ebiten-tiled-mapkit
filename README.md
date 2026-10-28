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

### Coordinates

There are three kinds of position, and each conversion lives where its knowledge is:

```
screen pixels  ◄── renderer, camera ──►  world pixels  ◄── map, tile size ──►  tiles
```

- **World pixels** (`tilemap.Vector`): where things are on the map, like `MapObject.Pos`.
- **Screen pixels**: where they appear, after the camera.
- **Tiles** (`tilemap.TilePos`): grid positions.

| From → to | Call |
|---|---|
| world → tile | `m.WorldToTile(pos)` |
| tile → world | `m.TileToWorld(tile)` (top-left corner) |
| world → screen | `renderer.WorldToScreen(pos, cam)` returns `x, y` |
| screen → world | `renderer.ScreenToWorld(x, y, cam)` |

```go
x, y := ebiten.CursorPosition()
hovered := m.WorldToTile(renderer.ScreenToWorld(x, y, cam)) // the tile under the mouse

standing := m.WorldToTile(player.Pos) // game logic needs no renderer
```

Map conversions work without a renderer, so game logic and tests only need the map.

### Objects

Object layers hold `MapObject`s: spawn points, triggers, items. They are descriptors; your game
creates its own entities from them. `Class` is the object's *Class* in Tiled (saved as `type` in
the .tmx file). A tile object without its own class takes its tile's class,
and its tile's properties under its own, as Tiled shows them.

```go
for o := range m.Objects() { // every object layer, groups included
	switch o.Class {
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

`m.WorldToTile(o.Center())` is the tile an object stands on, and `m.ObjectsAt(pos)` yields the
visible objects covering a tile, for example to select one with the mouse:

```go
x, y := ebiten.CursorPosition()
for o := range m.ObjectsAt(m.WorldToTile(renderer.ScreenToWorld(x, y, cam))) {
	if o.Class == "Door" { ... }
}
```

An object covers the tiles whose centers its bounds contain, so a 2x2 object is found on all four
and an object slightly off the grid on one. These look at map objects, which keep the positions
from Tiled: for objects your game moves, such as spawned characters, keep your own index.

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
| class | `tilemap.Properties` | `Props.Nested` |

Maps, layers, tiles and objects all have `Props`. Getters return the zero value for a missing
property. File paths are resolved like images, to paths within the map's file system, so they can
be opened from it directly. An object property holds the target's ID: `m.Object(o.Props.Int("target"))`.

## Drawing your own entities

Entities can use the map's tiles and animations, drawn between map layers:

```go
r.AfterLayer = func(dst *ebiten.Image, l tilemap.Layer, cam render.Camera) {
	if l.Base().Name != "objects" {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(r.WorldToScreen(player.pos, cam))
	r.DrawTile(dst, tilemap.Cell{TileID: player.tile, Flip: player.flip}, r.Clock()-player.animStart, op)

	// Any image works the same way, here half transparent.
	op = &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(0.5)
	op.GeoM.Translate(r.WorldToScreen(selected.pos, cam))
	dst.DrawImage(ui.Get("selector"), op)
}
```

- `AfterLayer` runs after every layer, visible or not, in draw order.
- `WorldToScreen` turns a world position into screen coordinates and plugs into
  `GeoM.Translate`, for tiles, images, text or shapes. Scale or rotate before translating to turn
  around the top-left corner.
- `DrawTile` draws a map tile with the cell's flips and the tile offset, then your `op`. Its
  top-left corner is at the origin of `op.GeoM`, in screen pixels, so it works for HUDs too.
- The time argument picks the animation frame. `r.Clock()` keeps the tile in sync with the map,
  and `r.Clock() - start` plays the animation from its first frame since `start`. Restarting,
  pausing or changing speed is just changing that number.

`m.Frame(id, t)` is the tile an animated tile shows at time `t`, or `id` itself when it isn't
animated, for example to read a property of the current frame. For play-once animations (a door
opening, a box landing), `m.FrameOnce(id, t)` holds the last frame and reports when it's done.

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
| `render` | `MapRenderer`, `ImageCache`, `Images`. |

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
