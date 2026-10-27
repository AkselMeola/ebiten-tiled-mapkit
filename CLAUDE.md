# CLAUDE.md

Go library: Tiled map loading and Ebiten rendering. Module `github.com/AkselMeola/ebiten-tiled-mapkit`.

Read `docs/DESIGN.md` before changing the domain model, an adapter or the renderer. It holds the
architecture, the decisions behind it and the roadmap. Update it when a decision changes.

Planned work is in `TODO.md`. Tick items off as they land.

## Invariants

- `tilemap` imports nothing from this module, Ebiten or Tiled. go-tiled types never leave `tiled`.
- `MapObject.Pos` is the top-left corner, rotation around it, for every adapter. Format-specific
  anchors are converted in the adapter.
- Animation is a function of time. No per-tile state, no wall-clock time in `Draw`.
- Adapters never load pixels; `render` does, through a `TextureProvider`.
- The library holds no game logic. Game-specific behavior belongs in the example or the game.
- Isometric is planned (`TODO.md`). Keep screen placement, culling, tile iteration and depth
  sorting behind `Projection`; don't add code that assumes world pixels are screen pixels.
- Stay lightweight: map loading and drawing only. Prefer a small hook or helper over a framework.

## Commands

```bash
go test ./...
go vet ./...
go test ./render -run xxx -bench . -count 5   # renderer benchmark; needs a display
go run ./examples/coins
cd examples/coins/assets && go run gen.go   # regenerate the example tileset
```

## Style

- Comments: very brief, or none. Don't explain what the code says.
- Tests use `testing/fstest.MapFS` with inline map files, plus the example map for a real-file check.
- Keep `README.md` (users) and `docs/DESIGN.md` (contributors) in sync with API changes.
