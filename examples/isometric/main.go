// Command isometric walks a player around an isometric Tiled map.
//
//	go run ./examples/isometric
//
// The arrows move along the map's grid axes: Up and Down change the row, Left and Right the
// column. On screen that is diagonal: Up goes up-right, Left goes up-left.
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

//go:embed map.tmx Tiles.tsx grass.png player.png tree.png wall.png
var assets embed.FS

const (
	screenWidth  = 960
	screenHeight = 540
	// walkTicks is how many ticks one step takes (Ebiten runs 60 ticks per second).
	walkTicks = 16
)

// Player is the game's own entity, created from the map's Player object. It is a render.Sprite:
// the renderer draws it among the trees and walls, in front of or behind them by where it stands.
type Player struct {
	tile     tilemap.TilePos // the tile the player is on, or walking to
	position tilemap.Vector  // where the player stands, in world pixels
	cell     tilemap.Cell    // the map tile used as the player's image
	// imageOffset moves the image's top-left from the player's feet on screen.
	imageOffset tilemap.Vector
	renderer    *render.MapRenderer
}

// Foot is where the player stands. The renderer sorts sprites and tiles by it.
func (p *Player) Foot() tilemap.Vector {
	return p.position
}

// Draw draws the player with its feet at screen position x, y, which the renderer projected from
// Foot with the camera applied.
func (p *Player) Draw(screen *ebiten.Image, x, y float64) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(x+p.imageOffset.X, y+p.imageOffset.Y)
	p.renderer.DrawTile(screen, p.cell, p.renderer.Clock(), options)
}

type Game struct {
	tileMap  *tilemap.Map
	renderer *render.MapRenderer
	player   Player
	// sprites is what the game draws in the Standing layer, reused every frame.
	sprites []render.Sprite
}

func main() {
	tileMap, err := tiled.Load(assets, "map.tmx")
	if err != nil {
		log.Fatal(err)
	}
	// The renderer picks the isometric projection from the map's orientation.
	renderer, err := render.NewMapRenderer(render.NewImageCache(assets), tileMap)
	if err != nil {
		log.Fatal(err)
	}

	game := &Game{tileMap: tileMap, renderer: renderer}
	if err := game.spawnPlayer(); err != nil {
		log.Fatal(err)
	}
	// The trees and walls are in the Standing layer. The player joins it, so the renderer sorts
	// them all by depth. Flat ground stays in its own layer below, so it never covers the player.
	game.sprites = []render.Sprite{&game.player}
	renderer.Sprites = game.spritesIn

	ebiten.SetWindowTitle("mapkit example: isometric")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// spawnPlayer turns the map's Player object into the game's player and hides the object, so the
// renderer doesn't draw it a second time.
func (g *Game) spawnPlayer() error {
	for object := range g.tileMap.Objects() {
		if object.Class != "Player" {
			continue
		}
		// ObjectTile is the tile Tiled shows the object on, whatever the map's orientation.
		tile := g.tileMap.ObjectTile(object)
		// The image is drawn like the map's tiles: its bottom edge on the bottom corner of the
		// tile's diamond, half a tile below where the player stands, so shift it left by half the
		// width and up by the full height. The artwork leaves room below the feet for that.
		imageSize := g.tileMap.TileDef(object.Cell.TileID).Src.Size()
		imageOffset := tilemap.Vector{
			X: -float64(imageSize.X) / 2,
			Y: float64(g.tileMap.TileHeight)/2 - float64(imageSize.Y),
		}
		g.player = Player{tile: tile, position: g.tileCenter(tile), cell: object.Cell, imageOffset: imageOffset, renderer: g.renderer}
		object.Visible = false
		return nil
	}
	return fmt.Errorf("map has no Player object")
}

// tileCenter is the middle of a tile's diamond in world pixels. World space is the same in every
// orientation (col × TileWidth, row × TileHeight); only drawing projects it to the screen.
func (g *Game) tileCenter(tile tilemap.TilePos) tilemap.Vector {
	half := tilemap.Vector{X: float64(g.tileMap.TileWidth) / 2, Y: float64(g.tileMap.TileHeight) / 2}
	return g.tileMap.TileToWorld(tile).Add(half)
}

func (g *Game) Update() error {
	g.renderer.Update()
	player := &g.player

	// Slide toward the target tile, one walkTicks-th of a tile per tick.
	target := g.tileCenter(player.tile)
	step := tilemap.Vector{X: float64(g.tileMap.TileWidth) / walkTicks, Y: float64(g.tileMap.TileHeight) / walkTicks}
	player.position = tilemap.Vector{
		X: moveToward(player.position.X, target.X, step.X),
		Y: moveToward(player.position.Y, target.Y, step.Y),
	}

	// Take the next step once the player has arrived, so movement is tile by tile.
	if player.position == target {
		next := player.tile.Add(readInput())
		if !g.isBlocked(next) {
			player.tile = next
		}
	}
	return nil
}

// readInput returns one grid step for the pressed arrow key.
func readInput() tilemap.TilePos {
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyArrowUp):
		return tilemap.Up
	case ebiten.IsKeyPressed(ebiten.KeyArrowDown):
		return tilemap.Down
	case ebiten.IsKeyPressed(ebiten.KeyArrowLeft):
		return tilemap.Left
	case ebiten.IsKeyPressed(ebiten.KeyArrowRight):
		return tilemap.Right
	}
	return tilemap.TilePos{}
}

func moveToward(current, target, speed float64) float64 {
	if current < target {
		return min(current+speed, target)
	}
	return max(current-speed, target)
}

// isBlocked reports whether a tile can't be entered: off the map, or a tile of class "Wall"
// (the walls and the trees) in any tile layer.
func (g *Game) isBlocked(tile tilemap.TilePos) bool {
	if !g.tileMap.InBounds(tile) {
		return true
	}
	for layer := range g.tileMap.TileLayers() {
		if tileDef := g.tileMap.TileDef(layer.At(tile).TileID); tileDef != nil && tileDef.Class == "Wall" {
			return true
		}
	}
	return false
}

func (g *Game) Draw(screen *ebiten.Image) {
	// The camera is a position on the whole map's image. WorldToScreen with a zero camera gives
	// the player's place on that image; centering the screen on it follows the player.
	playerX, playerY := g.renderer.WorldToScreen(g.player.position, render.Camera{})
	camera := render.Camera{X: playerX - screenWidth/2, Y: playerY - screenHeight/2}

	g.renderer.Draw(screen, camera)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("tile %d,%d  arrows move", g.player.tile.Col, g.player.tile.Row))
}

// spritesIn is the renderer's Sprites hook: the game's sprites for each layer.
func (g *Game) spritesIn(layer tilemap.Layer) []render.Sprite {
	if layer.Base().Name == "Standing" {
		return g.sprites
	}
	return nil
}

func (g *Game) Layout(int, int) (int, int) {
	return screenWidth, screenHeight
}
