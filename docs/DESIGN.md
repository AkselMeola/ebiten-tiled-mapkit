# Design

mapkit loads tile maps through pluggable adapters into a format-agnostic domain model, which a
renderer draws with Ebiten through a swappable projection.

```
Tiled adapter ─┐
(Text adapter) ┴─► tilemap.Map (pure data) ─► render.MapRenderer (Ebiten)
                                                ├─ Projection (orthogonal; isometric later)
                                                ├─ animation clock
                                                └─ TextureProvider
```

## Goals

- Load Tiled maps and parse tile layers, tiles, tileset images, objects and custom properties.
- Keep the domain independent of Tiled, so other adapters (text, JSON) can produce the same `Map`.
- Render a `Map` with swappable projections.
- Animate tiles with a shared clock, and let games animate their own entities with the same data.

## Packages and dependency rule

```
tilemap/   domain model; no Ebiten, no Tiled imports
tiled/     adapter: .tmx -> tilemap.Map
render/    Ebiten renderer, projections, textures, named images
examples/  example games
```

- `tiled` (and future adapters) import `tilemap`.
- `render` imports `tilemap` and Ebiten.
- `tilemap` imports nothing from this module, Ebiten or Tiled.

`github.com/lafriks/go-tiled` is an implementation detail of `tiled`; its types never appear in
`tilemap` or in the public API of `tiled`.

## Domain model (`tilemap`)

The domain refers to tiles by its own `TileID`, a dense index into `Map.Tiles`. Tiled concepts such
as GIDs, `firstgid` and local tile ids exist only inside `tiled`.

```go
type TileID uint32 // 0 = empty cell; Tiles[0] is a reserved sentinel

type Flip uint8 // FlipH | FlipV | FlipD (diagonal; with H/V gives 90° rotations)

type Cell struct {
    Tile TileID
    Flip Flip
}

type TileDef struct {
    Image     ImageRef        // slash path inside the map's fs.FS; no pixels
    Src       image.Rectangle // region of that image
    Offset    image.Point     // drawing offset (Tiled tileoffset)
    Class     string          // user tile class, e.g. "Wall"
    Animation []Frame         // nil if static; set via SetAnimation to cache the total duration
    Props     Properties
    Colliders []Shape         // relative to the tile's top-left
}

type Map struct {
    Width, Height         int // tiles
    TileWidth, TileHeight int // pixels
    Orientation           Orientation // a hint; the renderer picks the projection
    Tiles                 []TileDef   // index = TileID
    Layers                []Layer     // ordered; also the draw order
    Props                 Properties
}
```

### Layers

`Layer` is a closed interface (unexported marker method) so one ordered slice holds every kind.
Common fields live in an embedded `LayerBase` (`Name`, `Class`, `Visible`, `Opacity`, offsets,
`Props`). Kinds: `*TileLayer` (flat row-major `[]Cell`, never one object per tile), `*ObjectLayer`,
`*ImageLayer`, `*GroupLayer`. Groups are kept, not flattened; their visibility, opacity and offset
apply to their children at draw time.

### Objects

`MapObject` (Tiled's term) is deliberately not called Entity or Sprite. Objects are spawn
descriptors: the game reads them and creates its own entities, so game logic never leaks into the
library.

```go
type MapObject struct {
    ID         int
    Name, Type string
    Pos, Size  Vector  // Pos = top-left corner
    Rotation   float64 // degrees, clockwise, around the top-left corner
    Visible    bool
    Opacity    float64
    Tile       Cell    // zero if not a tile object
    Shape      Shape   // rect, ellipse, point, polygon, polyline
    Props      Properties
}
```

**Anchor normalization.** Tiled anchors tile objects at the bottom-left (or per the tileset's
`objectalignment`, or bottom-center on isometric maps). The domain has one rule for every object:
`Pos` is the top-left corner and rotation turns around it. The Tiled adapter converts, including for
rotated objects; the conversion is lossless. `Bounds()` returns the rotated AABB and `SortY()` its
bottom edge for depth sorting. `Center()` and `Foot()` (bottom center) also follow rotation.

**Vectors.** Pixel positions and sizes are `Vector{X, Y float64}` with `Add`, `Sub`, `Scale` and
`Rotate`. It stays a plain two-field struct so it compares with `==` and converts directly to a
game's own vector type with the same fields. Tile coordinates stay `col, row int` and pixel
rectangles stay `image.Rectangle`.

### Properties

`Properties` is `map[string]any` holding typed values: `string`, `int`, `float64`, `bool`,
`color.RGBA`, `FilePath`, `ObjectRef`, or nested `Properties` for class properties. Typed getters
return zero values when missing.

### Zero values

`Visible` and `Opacity` are plain fields, so their zero values mean hidden and transparent. Adapters
must set them explicitly.

## Tiled adapter (`tiled`)

`tiled.Load(fsys fs.FS, name string)` reads the map from `fsys`; tilesets, templates and images are
resolved relative to it inside the same `fsys`, so `embed.FS` and `os.DirFS` both work.

Steps:

1. Read the file once; scan it for the layer order and the `infinite` flag (see below).
2. Parse with go-tiled.
3. For each tileset, for each tile, append a `TileDef` and record `gidToTile[firstGID+i]`.
   Atlas tilesets get one def per tile index with its source rect; image-collection tilesets get one
   per `<tile>` with an image (sparse ids are fine).
4. Translate animation frames through `gidToTile` once all tilesets are known.
5. Convert layers: strip flip bits from GIDs into `Cell.Flip`, look up `TileID`.
6. Convert objects: merge templates, resolve tile objects, normalize the anchor, convert shapes and
   properties.

go-tiled gaps worked around in the adapter:

- **Layer order.** go-tiled keeps each layer kind in a separate slice, losing the interleaved order.
  `layout.go` recovers it with a light XML token scan.
- **Lazy tilesets.** go-tiled only initializes tilesets referenced by a GID; the adapter initializes
  the rest (external ones are loaded, embedded ones get their base dir).
- **Points.** go-tiled doesn't parse `<point/>`; a sizeless object with no other geometry is treated
  as a point.
- **Templates.** go-tiled loads templates but doesn't merge them. The instance's own fields win; a
  template tile GID is remapped to the map tileset with the same resolved source path.

Not supported yet: `.tmj`, infinite maps (rejected with `ErrInfiniteMap`), zstd, tileset `trans`
color, text objects (loaded as rects), parallax, tint, image layer repeat.

## Animation

Tile animation is a pure function of time; tiles hold no per-instance state.

```go
func (d *TileDef) FrameAt(t time.Duration) TileID                    // loops; 0 for static tiles
func (d *TileDef) FrameAtOnce(t time.Duration) (id TileID, done bool) // plays once, holds last frame
```

`FrameAt` walks `t % total` through the frame durations. All instances of a tile stay in sync, as in
Tiled, and advancing animation costs one clock increment per tick.

| Clock | `Clock % 600ms` | Frame (`[{4,200ms},{5,200ms},{6,200ms}]`) |
|---|---|---|
| 0ms | 0 | 4 |
| 250ms | 250 | 5 |
| 500ms | 500 | 6 |
| 650ms | 50 | 4 |

The clock is the renderer's: `MapRenderer.Update()` adds `render.Tick()` and `Clock()` reads it.
Call `Update` from Ebiten's `Update`, never wall-clock time in `Draw`, so animation is
deterministic. A new level gets a new renderer, so its clock starts at zero.

**Per-entity animation** needs no extra machinery: an entity stores when its current animation
started and passes `clock - start` as the time. That restarts, desyncs, slows down (`* speed`) or
pauses an animation. This replaced a per-object player map once planned on a shared animator: the
state lives on the game entity, which outlives the object it was spawned from.

## Renderer (`render`)

```go
type Projection interface {
    TileToScreen(col, row int) (x, y float64)
    ScreenToTile(x, y float64) (col, row int)
    Bounds(m *tilemap.Map) image.Rectangle
    VisibleTiles(m *tilemap.Map, view image.Rectangle) (c0, r0, c1, r1 int) // inclusive
}

type TextureProvider interface {
    Texture(ref tilemap.ImageRef) (*ebiten.Image, error)
}

type MapRenderer struct {
    AfterLayer LayerHook // runs after every layer, visible or not
    // map, projection, textures, clock and caches are private
}

func NewMapRenderer(textures TextureProvider, m *tilemap.Map) (*MapRenderer, error)
func (r *MapRenderer) Update()
func (r *MapRenderer) Clock() time.Duration
func (r *MapRenderer) Projection() Projection
func (r *MapRenderer) Draw(dst *ebiten.Image, cam Camera)
func (r *MapRenderer) DrawTile(dst *ebiten.Image, cell tilemap.Cell, t time.Duration, op *ebiten.DrawImageOptions)
```

**One renderer per map.** The constructor picks the projection from `Map.Orientation` (orthogonal,
or unknown for hand-built maps; others are an error until implemented), loads every tile and image
layer image and returns all errors, and cuts the tile sub-images. Nothing can go stale: there is no
`SetMap`, and a renderer is never half set up. Switching levels means a new renderer; decoded
images live in the `TextureProvider`, which the game keeps for its whole run and shares with
`Images`.

Draw walks `Map.Layers` in order with a type switch, skipping invisible layers and accumulating
group offsets and opacity. The camera is rounded to whole pixels first, and `AfterLayer` gets
the rounded camera so entities stay aligned with the map. For a tile layer it culls to the visible
range and adds one quad per cell to a vertex batch: resolve animation, anchor bottom-left in the
cell (tiles larger than the grid stick out upward, as in Tiled), add tile offset, layer offset and
camera, and flip the corners as `DrawTile` does. Object layers draw only tile objects; other
objects have no visuals. An object layer's `DrawOrder` is `DrawOrderTopDown` (sorted by `SortY()`,
stable; the zero value and Tiled's default) or `DrawOrderIndex` (file order). Sorting happens every
frame so objects the game moves stay correct; the index buffer is reused.

`DrawTile` exposes the same tile drawing for game entities: flips and tile offset first, then the
caller's `op` (position, rotation, scale, color). Entities are drawn from the `AfterLayer` hook so
they interleave with map layers.

Implementation notes:

- **Sub-image cache.** `SubImage` allocates; tile images are cached in a `[]*ebiten.Image` indexed by
  `TileID`, built by the constructor.
- **Textures via `fs.FS`.** `FSTextures` loads and caches images, errors included. Adapters never
  touch pixels, so they run headless in tests. `Images` names images and sheet regions outside the
  map (UI) on top of a `TextureProvider`, sharing its cache; regions are checked against the image.
- **Culling margins.** Tiles can extend outside their cell (larger than the grid, or offset). The
  renderer computes how far once per map and widens the view before calling `VisibleTiles`.
- **Tile layer batching.** One `DrawTriangles32` call per run of tiles sharing a tileset image,
  so a single-tileset layer is one call; a texture change flushes the batch, keeping draw order.
  The vertex and index slices are reused between frames. `BenchmarkDrawFullHD` (25k tiles):
  ~8.2 ms with one `DrawImage` per tile, ~2.4 ms batched.
- **Whole pixels.** `DrawImage` snaps vertices to a third of a pixel; `DrawTriangles32` doesn't,
  so at fractional positions the edge pixels sample outside the tile and neighboring tiles in the
  tileset bleed in. Quads are placed on whole pixels.
- **Flips.** One implementation, `applyFlips` on a `GeoM`: diagonal first (transpose), then
  horizontal, then vertical, within the tile's bounds. Batched quads apply it to their corners.
- **Not batched.** Tile objects (few, rotated, scaled, sorted), image layers and `DrawTile` use
  `DrawImage`. Pre-rendering static layers only if profiling asks for it.

## Load and run lifecycle

At startup: `FSTextures`, and `Images` for UI graphics.
Per level: adapter → `tilemap.Map`, then `NewMapRenderer(textures, m)`.
Per frame: `renderer.Update()` in `Update`, `renderer.Draw(screen, cam)` in `Draw`.

## Roadmap

Done: domain model; Tiled adapter (CSV/XML/base64 + zlib/gzip, embedded and external tilesets, atlas
and image-collection tilesets, animations, flips, offsets, colliders, objects, templates, group and
image layers, properties); orthogonal projection with culling; shared-clock animation; flips;
object layers and tile objects; layer opacity and offsets; `DrawTile` for entities.

Next, roughly in order:

1. Isometric projection, which validates the `Projection` abstraction, then draw order as a strategy
   rather than a subclass.
2. Text adapter, to validate that the domain is truly format-agnostic.
3. `.tmj` maps; infinite maps (normalize chunks to one grid on load); zstd.
4. Worlds (below).

### Worlds

A Tiled `.world` file groups maps in a shared coordinate space. It sits above `Map`:

```go
type World struct{ Maps []PlacedMap }

type PlacedMap struct {
    Map    *Map        // or a loader for lazy loading
    Offset image.Point // world pixels
}
```

`tiled.LoadWorld` would produce it; the renderer draws each visible map with its offset added to
the camera, culling whole maps outside the view. This doesn't change `Map`.

### Deferred

- Staggered and hexagonal projections.
- Pre-rendering static layers (until profiling asks for it).
- Fallback rendering for tiles without an image (for the text adapter).
