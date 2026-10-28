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
- [ ] `MapRenderer.DrawObject(dst, m, obj, cam)`: draws one map object exactly as the renderer does
      (size, rotation, opacity, camera), so games can y-sort map objects together with entities.
- [x] `MapObject.Foot()`: bottom-center point, rotation applied, for spawning characters where
      they stand.
- [ ] Coins example: a tree as a tile object that the player can walk behind (y-sorted in
      `AfterLayer` of a hidden object layer), trunk blocking.
- [ ] Document the y-sorting pattern in README.

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
Decide before building it:

- [ ] What world space is in isometric maps. Option: the adapter normalizes Tiled's tile-axis
      coordinates to column × `TileWidth`, row × `TileHeight`, so `tilemap` stays orientation-free
      and only `WorldToScreen`/`ScreenToWorld` project.
- [ ] Tile objects: `Pos` as world top-left doesn't map to the screen-aligned image. Option: draw
      from `WorldToScreen(o.Foot())` shifted by (−w/2, −h). Settle with an adapter test against Tiled.
- [ ] Make `MapObject.SortY()` internal: depth belongs to the projection.
- [ ] Rename the internal `projection.tileToWorld` to `tileToScreen`; it differs from
      `Map.TileToWorld` in isometric maps.
- [ ] Document `Camera`'s space (screen pixels of the projected map); maybe make it a `Vector`.


- [ ] `isometric` projection: `tileToWorld` `((c−r)·w/2, (c+r)·h/2)`, `visibleTiles`.
- [ ] Culling: the view is a diamond in tile space. Let the projection drive tile iteration instead
      of the renderer looping over a `c0..c1 × r0..r1` rectangle.
- [ ] Tile image anchor from the projection (Tiled centers images horizontally in isometric maps).
- [ ] `MapRenderer.WorldToScreen` and `ScreenToWorld` through the projection: isometric object
      positions are in tile-axis space, not screen pixels. `drawObjectLayer` already uses
      `WorldToScreen`; also `DrawObject` and y-sorting (depth is the foot's screen y, not `SortY()`).
- [ ] Tiled adapter: tile object anchor is bottom-center in isometric maps.
- [ ] `Map.WorldToTile`, `TileToWorld` and `ObjectsAt`: isometric object positions are in
      tile-axis space.
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
