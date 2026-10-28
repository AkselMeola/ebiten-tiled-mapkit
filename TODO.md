# TODO

## v0.2.0: depth and objects

- [x] Rename `tilemap.Point` to `tilemap.Vector` (`struct{ X, Y float64 }`) with `Add`, `Sub` and
      `Scale`. Breaking change.
- [x] `MapObject`: `X, Y, W, H` → `Pos, Size Vector`; same for `Shape`; polygon points become
      `[]Vector`. Tile coordinates are `TilePos`, pixel rectangles stay `image.Rectangle`.
- [x] New helpers (`Foot()`, `Center()`) return `Vector`. Keep the field layout identical so games
      can convert to their own vector type (`grid.Vector(o.Pos)`).
- [x] Update README, `docs/DESIGN.md`, the coins example and the kastimees sandbox.

- [x] Object layer draw order: read Tiled's `draworder` into `ObjectLayer`. With `topdown` (Tiled's
      default) the renderer sorts by `SortY()`; with `index` it keeps file order.
- [x] ~~`MapRenderer.DrawObject(dst, m, obj, cam)`~~: not needed, `Sprites` sorts entities into
      the object layer with its tile objects.
- [x] `MapObject.Foot()`: bottom-center point, rotation applied, for spawning characters where
      they stand.
- [ ] Coins example: a tree as a tile object that the player can walk behind (the player is
      already a sprite in the object layer), trunk blocking.
- [x] Document the y-sorting pattern in README.

## Tutorial

- [ ] `docs/TUTORIAL.md`: build a small game step by step, ending close to `examples/coins`.
  - [ ] Make a map in Tiled: tileset, ground and wall layers, an object layer with a player and coins.
  - [ ] Load and draw it (`ImageCache`, `tiled.Load`, `MapRenderer`).
  - [ ] Player entity from the spawn object, tile movement, wall collision via tile class.
  - [ ] Walk animation: idle/walk tiles found by properties, local animation time, flip by direction.
  - [ ] Collect coins: hide objects on pickup, count in a HUD.
  - [ ] Camera: follow the player, clamp to map bounds, smooth follow, mouse/keyboard pan.
  - [ ] Zoom: `Layout`-based zoom vs. an offscreen image, pixel-perfect integer zoom, zooming
        around the cursor, converting mouse position to world and tile coordinates.
- [ ] Link it from README; keep its code compiling (e.g. per-step examples under
      `examples/tutorial/`).

## API review, open

- [ ] `Shape` repeats `MapObject`'s `Pos`, `Size` and `Rotation` (zero or copied on map objects).
      Option: `Kind` and `Points` on `MapObject`, `TileDef.Colliders []MapObject`, drop `Shape`.
- [x] `TileDef.FrameAt` returned 0 for static tiles, so callers fell back to the tile id
      themselves. Now `Map.Frame(id, t)` and `Map.FrameOnce(id, t)`.

## Isometric

- [x] World space is column × `TileWidth`, row × `TileHeight` in every orientation; the Tiled
      adapter scales isometric object x. `WorldToTile`, `TileToWorld` and `ObjectsAt` stay as they are.
- [x] `isometric` projection: `tileToScreen`, `worldToScreen`, `screenToWorld`, `visibleTiles`,
      `depth`; `WorldToScreen` and `ScreenToWorld` go through it. Tile images sit bottom-left in
      the diamond's bounding box, as in Tiled.
- [x] Tile objects hang from `WorldToScreen(o.Foot())` in both projections; isometric object
      layers sort by the foot's screen y. `Map.ObjectTile(o)`: an isometric tile object stands on
      the tile whose bottom corner is nearest its foot.
- [x] `examples/isometric`: walk the grid with the arrows.
- [x] Depth sorting of entities with tall tiles: `render.Sprite` (`Foot`, `Draw`) and the
      `Sprites` hook sort the game's entities into a layer by the projection's depth, in both
      orientations. The example map keeps ground and standing tiles in separate layers.
- [ ] Sprites ignore layer opacity, and depth doesn't span layers. Revisit if a map needs either.
- [ ] Culling draws the tile rectangle around the view, about twice the visible diamond. Let the
      projection drive tile iteration if profiling asks for it.
- [ ] Non-default `objectalignment` on isometric maps: images still hang from the foot.
- [ ] Make `MapObject.SortY()` internal: depth belongs to the projection.
- [ ] Document `Camera` further; maybe make it a `Vector`.
- [ ] Staggered and hexagonal: own projections, later.

## Performance

- [x] Benchmark: full-HD view, 16 px tiles, three layers (`BenchmarkDrawFullHD`). Baseline on an
      i7-1260P: ~8 ms per `Draw` for ~25k tiles (~320 ns per tile), half a 60 FPS frame. Profile:
      ~98% of that is inside `ebiten.(*Image).DrawImage`, our own code ~2%.
- [x] ~~Reuse one `DrawImageOptions` per layer~~: measured, no gain. The options never escape to
      the heap (`go build -gcflags=-m`), so there was nothing to save. Not done.
- [x] One `DrawTriangles32` call per run of tiles sharing a texture instead of one `DrawImage` per
      tile: ~8.2 ms → ~2.4 ms in the benchmark, measured back to back. Pixel-identical for whole-
      pixel cameras; the camera is now rounded to whole pixels to avoid bleeding.
- [ ] Only if profiling asks for it: pre-render static (non-animated) layers into chunk images.
