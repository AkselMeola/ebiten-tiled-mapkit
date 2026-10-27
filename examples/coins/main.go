// Command coins is a small example game: walk around a Tiled map and collect every coin.
//
//	go run ./examples/coins
//
// Arrows or WASD move.
package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/AkselMeola/ebiten-tiled-mapkit/render"
	"github.com/AkselMeola/ebiten-tiled-mapkit/tiled"
	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// The map, its tileset and the tileset image are embedded, so the game is a single binary.
// tiled.Load resolves the tileset and image paths relative to the map inside this filesystem.
//
//go:embed assets/coins.tmx assets/tiles.tsx assets/tiles.png
var assets embed.FS

const (
	// windowScale enlarges the small pixel-art map in the window.
	windowScale = 2
	// walkSpeed is in pixels per tick (Ebiten runs 60 ticks per second). It divides the tile
	// size evenly, so the player always stops exactly on a tile.
	walkSpeed = 4
)

// Player is the game's own entity. The map only says where it starts; from then on the game
// owns its position, direction and drawing.
type Player struct {
	position tilemap.Vector // current position in pixels, top-left corner
	target   tilemap.Vector // the tile the player is walking to
	tile     tilemap.TileID
	flip     tilemap.Flip // mirrored when facing left
}

// isWalking reports whether the player is between two tiles.
func (p *Player) isWalking() bool {
	return p.position != p.target
}

type Game struct {
	tileMap     *tilemap.Map
	groundLayer *tilemap.TileLayer
	objectLayer *tilemap.ObjectLayer
	renderer    *render.MapRenderer
	animator    *render.Animator

	player         Player
	coinCount      int
	collectedCount int
}

func main() {
	tileMap, err := tiled.Load(assets, "assets/coins.tmx")
	if err != nil {
		log.Fatal(err)
	}

	game := &Game{
		tileMap:     tileMap,
		groundLayer: tileMap.Layer("ground").(*tilemap.TileLayer),
		objectLayer: tileMap.Layer("objects").(*tilemap.ObjectLayer),
		animator:    &render.Animator{},
	}

	// The renderer draws the map. The animator's clock drives the animated tiles (water, coins).
	game.renderer = render.NewMapRenderer(render.NewOrthogonal(tileMap), render.NewFSTextures(assets), game.animator)
	// AfterLayer runs after each map layer is drawn; we use it to draw the player between layers.
	game.renderer.AfterLayer = game.drawPlayer
	// Preload loads every tileset image now, so a missing file fails at startup, not mid-game.
	if err := game.renderer.Preload(tileMap); err != nil {
		log.Fatal(err)
	}

	game.spawnFromObjects()

	ebiten.SetWindowTitle("mapkit example: coins")
	ebiten.SetWindowSize(tileMap.Width*tileMap.TileWidth*windowScale, tileMap.Height*tileMap.TileHeight*windowScale)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// spawnFromObjects reads the objects placed in Tiled. Objects are only descriptions: the game
// decides what each one becomes.
func (g *Game) spawnFromObjects() {
	for i := range g.objectLayer.Objects {
		object := &g.objectLayer.Objects[i]
		switch object.Type {
		case "Player":
			// The player becomes a game entity. Hide the map object so it isn't drawn twice.
			g.player = Player{position: object.Pos, target: object.Pos, tile: object.Tile.Tile}
			object.Visible = false
		case "Coin":
			// Coins stay on the map; the renderer keeps drawing and animating them.
			g.coinCount++
		}
	}
}

func (g *Game) Update() error {
	// Advance the shared clock by one tick. All map animations are computed from it.
	g.animator.Update(render.Tick())

	player := &g.player

	// Slide toward the target a few pixels per tick.
	player.position = tilemap.Vector{
		X: moveToward(player.position.X, player.target.X),
		Y: moveToward(player.position.Y, player.target.Y),
	}

	// Input is only read once on a tile, so the player moves tile by tile. Reading it right after
	// arriving keeps a held key walking without a standing frame in between.
	if !player.isWalking() {
		g.collectCoins()

		direction := readInput()
		if direction.X < 0 {
			player.flip = tilemap.FlipH
		} else if direction.X > 0 {
			player.flip = 0
		}

		// One tile further in the pressed direction.
		step := tilemap.Vector{X: direction.X * float64(g.tileMap.TileWidth), Y: direction.Y * float64(g.tileMap.TileHeight)}
		next := player.position.Add(step)
		if !g.isBlocked(next) {
			player.target = next
		}
	}
	return nil
}

// readInput returns the pressed direction as -1, 0 or 1 per axis. Only one direction at a time.
func readInput() tilemap.Vector {
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA):
		return tilemap.Vector{X: -1}
	case ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD):
		return tilemap.Vector{X: 1}
	case ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW):
		return tilemap.Vector{Y: -1}
	case ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS):
		return tilemap.Vector{Y: 1}
	}
	return tilemap.Vector{}
}

// moveToward steps from current toward target by walkSpeed, without overshooting.
func moveToward(current, target float64) float64 {
	if current < target {
		return min(current+walkSpeed, target)
	}
	return max(current-walkSpeed, target)
}

// isBlocked reports whether the tile at the pixel position can't be entered. Walls are marked
// in Tiled by giving their tiles the class "Wall"; outside the map there is no tile at all.
func (g *Game) isBlocked(position tilemap.Vector) bool {
	column := int(position.X) / g.tileMap.TileWidth
	row := int(position.Y) / g.tileMap.TileHeight
	tileDef := g.tileMap.Tile(g.groundLayer.At(column, row).Tile)
	return tileDef == nil || tileDef.Class == "Wall"
}

// collectCoins picks up a coin the player is standing on. Coins and the player both sit
// exactly on tiles, so comparing positions is enough. A collected coin is just hidden.
func (g *Game) collectCoins() {
	for i := range g.objectLayer.Objects {
		object := &g.objectLayer.Objects[i]
		if object.Type == "Coin" && object.Visible && object.Pos == g.player.position {
			object.Visible = false
			g.collectedCount++
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draws every layer in order; the player is drawn in between by drawPlayer.
	// The camera stays at the origin because the whole map fits in the window.
	g.renderer.Draw(screen, g.tileMap, render.Camera{})

	status := fmt.Sprintf("coins %d/%d", g.collectedCount, g.coinCount)
	if g.collectedCount == g.coinCount {
		status += "  - all collected!"
	}
	ebitenutil.DebugPrint(screen, status)
}

// drawPlayer is the renderer's AfterLayer hook. Drawing right after the object layer puts the
// player above the ground and next to the coins.
func (g *Game) drawPlayer(screen *ebiten.Image, layer tilemap.Layer, camera render.Camera) {
	if layer != g.objectLayer {
		return
	}
	player := &g.player

	// The time picks the animation frame: the map clock while walking, frame one when standing.
	animationTime := g.animator.Clock
	if !player.isWalking() {
		animationTime = 0
	}

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(player.position.X-camera.X, player.position.Y-camera.Y)
	// DrawTile draws a map tile for the game's own entity, with flips and animation applied.
	g.renderer.DrawTile(screen, g.tileMap, tilemap.Cell{Tile: player.tile, Flip: player.flip}, animationTime, options)
}

// Layout makes the game's screen exactly the map's size; Ebiten scales it up to the window.
func (g *Game) Layout(int, int) (int, int) {
	return g.tileMap.Width * g.tileMap.TileWidth, g.tileMap.Height * g.tileMap.TileHeight
}
