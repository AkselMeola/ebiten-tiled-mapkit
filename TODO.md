# TODO

## v0.2.0: depth and objects

- [ ] Rename `tilemap.Point` to `tilemap.Vector` (`struct{ X, Y float64 }`) with `Add`, `Sub` and
      `Scale`. Breaking change.
- [ ] `MapObject`: `X, Y, W, H` → `Pos, Size Vector`; same for `Shape`; polygon points become
      `[]Vector`. Tile coordinates stay `col, row int`, pixel rectangles stay `image.Rectangle`.
- [ ] New helpers (`Foot()`, `Center()`) return `Vector`. Keep the field layout identical so games
      can convert to their own vector type (`grid.Vector(o.Pos)`).
- [ ] Update README, `docs/DESIGN.md`, the coins example and the kastimees sandbox.

- [ ] Object layer draw order: read Tiled's `draworder` into `ObjectLayer`. With `topdown` (Tiled's
      default) the renderer sorts by `SortY()`; with `index` it keeps file order.
- [ ] `MapRenderer.DrawObject(dst, m, obj, cam)`: draws one map object exactly as the renderer does
      (size, rotation, opacity, camera), so games can y-sort map objects together with entities.
- [ ] `MapObject.Foot()`: bottom-center point, rotation applied, for spawning characters where
      they stand.
- [ ] Coins example: a tree as a tile object that the player can walk behind (y-sorted in
      `AfterLayer` of a hidden object layer), trunk blocking.
- [ ] Document the y-sorting pattern in README.

## Tutorial

- [ ] `docs/TUTORIAL.md`: build a small game step by step, ending close to `examples/coins`.
  - [ ] Make a map in Tiled: tileset, ground and wall layers, an object layer with a player and coins.
  - [ ] Load and draw it (`tiled.Load`, `MapRenderer`, `Animator`).
  - [ ] Player entity from the spawn object, tile movement, wall collision via tile class.
  - [ ] Walk animation: idle/walk tiles found by properties, local animation time, flip by direction.
  - [ ] Collect coins: hide objects on pickup, count in a HUD.
  - [ ] Camera: follow the player, clamp to map bounds, smooth follow, mouse/keyboard pan.
  - [ ] Zoom: `Layout`-based zoom vs. an offscreen image, pixel-perfect integer zoom, zooming
        around the cursor, converting mouse position to world and tile coordinates.
- [ ] Link it from README; keep its code compiling (e.g. per-step examples under
      `examples/tutorial/`).

## Performance

- [ ] Reuse one `DrawImageOptions` per layer instead of allocating per tile and per object
      (`drawTileLayer`, `drawObjectLayer`).
- [ ] Benchmark: full-HD view, 16 px tiles, three layers. Record before/after numbers.
- [ ] Only if the benchmark asks for it: one `DrawTriangles` call per layer and texture.
- [ ] Only if profiling asks for it: pre-render static (non-animated) layers into chunk images.
