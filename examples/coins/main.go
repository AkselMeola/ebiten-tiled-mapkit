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
// owns its position, direction and drawing. It is a render.Sprite, drawn by the renderer among
// the coins in the object layer.
type Player struct {
	position tilemap.Vector // current position in pixels, top-left corner
	target   tilemap.Vector // the tile the player is walking to
	tile     tilemap.TileID
	flip     tilemap.Flip   // mirrored when facing left
	size     tilemap.Vector // one tile
	renderer *render.MapRenderer
}

// Foot is where the player stands: the bottom center of its tile. The renderer sorts the
// player with the coins by it.
func (p *Player) Foot() tilemap.Vector {
	return p.position.Add(tilemap.Vector{X: p.size.X / 2, Y: p.size.Y})
}

// Draw draws the player with its feet at screen position x, y, which the renderer projected
// from Foot with the camera applied.
func (p *Player) Draw(screen *ebiten.Image, x, y float64) {
	// The time picks the animation frame: the map clock while walking, frame one when standing.
	animationTime := p.renderer.Clock()
	if !p.isWalking() {
		animationTime = 0
	}

	// DrawTile puts the tile's top-left at the translation: half a tile left of the feet and a
	// full tile above them.
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(x-p.size.X/2, y-p.size.Y)
	// DrawTile draws a map tile for the game's own entity, with the flip and animation frame.
	p.renderer.DrawTile(screen, tilemap.Cell{TileID: p.tile, Flip: p.flip}, animationTime, options)
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

	player         Player
	coinCount      int
	collectedCount int
}

func main() {
	// The image cache lives as long as the game. Every level (and UI image) loads through it, so
	// each image file is decoded only once.
	cache := render.NewImageCache(assets)

	game := &Game{}
	if err := game.startLevel(cache, "assets/coins.tmx"); err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("mapkit example: coins")
	ebiten.SetWindowSize(game.tileMap.Width*game.tileMap.TileWidth*windowScale, game.tileMap.Height*game.tileMap.TileHeight*windowScale)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// startLevel loads a map and everything that belongs to it. Switching levels is calling it again.
func (g *Game) startLevel(cache render.ImageSource, path string) error {
	tileMap, err := tiled.Load(assets, path)
	if err != nil {
		return err
	}

	// The renderer draws this one map. It loads the map's images now, so a missing file fails
	// here, not mid-game. Its clock drives the animated tiles (water, coins).
	renderer, err := render.NewMapRenderer(cache, tileMap)
	if err != nil {
		return err
	}
	// Sprites gives the renderer the game's own entities per layer. It draws them in that layer,
	// sorted by depth with the layer's tiles or objects.
	renderer.Sprites = g.spritesIn

	*g = Game{
		tileMap:     tileMap,
		groundLayer: tileMap.Layer("ground").(*tilemap.TileLayer),
		objectLayer: tileMap.Layer("objects").(*tilemap.ObjectLayer),
		renderer:    renderer,
	}
	g.spawnFromObjects()
	return nil
}

// spawnFromObjects reads the objects placed in Tiled. Objects are only descriptions: the game
// decides what each one becomes.
func (g *Game) spawnFromObjects() {
	for object := range g.tileMap.Objects() {
		switch object.Class {
		case "Player":
			// The player becomes a game entity. Hide the map object so it isn't drawn twice.
			g.player = Player{
				position: object.Pos,
				target:   object.Pos,
				tile:     object.Cell.TileID,
				size:     tilemap.Vector{X: float64(g.tileMap.TileWidth), Y: float64(g.tileMap.TileHeight)},
				renderer: g.renderer,
			}
			object.Visible = false
		case "Coin":
			// Coins stay on the map; the renderer keeps drawing and animating them.
			g.coinCount++
		}
	}
}

func (g *Game) Update() error {
	// Advance the map's animation clock by one tick.
	g.renderer.Update()

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
	tilePos := g.tileMap.WorldToTile(position)
	tileDef := g.tileMap.TileDef(g.groundLayer.At(tilePos).TileID)
	return tileDef == nil || tileDef.Class == "Wall"
}

// collectCoins picks up a coin the player is standing on. Coins and the player both sit
// exactly on tiles, so comparing positions is enough. A collected coin is just hidden.
func (g *Game) collectCoins() {
	for i := range g.objectLayer.Objects {
		object := &g.objectLayer.Objects[i]
		if object.Class == "Coin" && object.Visible && object.Pos == g.player.position {
			object.Visible = false
			g.collectedCount++
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draws every layer in order, the player inside the object layer.
	// The camera stays at the origin because the whole map fits in the window.
	g.renderer.Draw(screen, render.Camera{})

	status := fmt.Sprintf("coins %d/%d", g.collectedCount, g.coinCount)
	if g.collectedCount == g.coinCount {
		status += "  - all collected!"
	}
	ebitenutil.DebugPrint(screen, status)
}

// spritesIn is the renderer's Sprites hook. The player goes into the object layer: above the
// ground, sorted with the coins.
func (g *Game) spritesIn(layer tilemap.Layer) []render.Sprite {
	if layer == g.objectLayer {
		return []render.Sprite{&g.player}
	}
	return nil
}

// Layout makes the game's screen exactly the map's size; Ebiten scales it up to the window.
func (g *Game) Layout(int, int) (int, int) {
	return g.tileMap.Width * g.tileMap.TileWidth, g.tileMap.Height * g.tileMap.TileHeight
}
