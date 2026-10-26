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
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/AkselMeola/ebiten-tiled-mapkit/render"
	"github.com/AkselMeola/ebiten-tiled-mapkit/tiled"
	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

//go:embed assets/coins.tmx assets/tiles.tsx assets/tiles.png
var assets embed.FS

const (
	scale    = 2
	stepTime = 150 * time.Millisecond
)

type player struct {
	col, row     int
	fromX, fromY float64
	x, y         float64
	stepStart    time.Duration
	moving       bool
	tile         tilemap.TileID
	flip         tilemap.Flip
	walkStart    time.Duration
}

type game struct {
	m        *tilemap.Map
	ground   *tilemap.TileLayer
	objects  *tilemap.ObjectLayer
	renderer *render.MapRenderer
	anim     *render.Animator

	player    player
	coins     []*tilemap.MapObject
	collected int
}

func main() {
	m, err := tiled.Load(assets, "assets/coins.tmx")
	if err != nil {
		log.Fatal(err)
	}

	g := &game{
		m:       m,
		ground:  m.Layer("ground").(*tilemap.TileLayer),
		objects: m.Layer("objects").(*tilemap.ObjectLayer),
		anim:    &render.Animator{},
	}
	g.renderer = render.NewMapRenderer(render.NewOrthogonal(m), render.NewFSTextures(assets), g.anim)
	g.renderer.AfterLayer = g.afterLayer
	if err := g.renderer.Preload(m); err != nil {
		log.Fatal(err)
	}
	g.spawn()

	ebiten.SetWindowTitle("mapkit example: coins")
	ebiten.SetWindowSize(m.Width*m.TileWidth*scale, m.Height*m.TileHeight*scale)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// spawn turns map objects into game state. The player becomes an entity drawn by the game;
// coins stay on the map and are hidden when collected.
func (g *game) spawn() {
	for i := range g.objects.Objects {
		o := &g.objects.Objects[i]
		switch o.Type {
		case "Player":
			col, row := g.cellOf(o)
			g.player = player{col: col, row: row, x: o.X, y: o.Y, tile: o.Tile.Tile}
			o.Visible = false
		case "Coin":
			g.coins = append(g.coins, o)
		}
	}
}

func (g *game) cellOf(o *tilemap.MapObject) (col, row int) {
	return int(o.X) / g.m.TileWidth, int(o.Y) / g.m.TileHeight
}

func (g *game) blocked(col, row int) bool {
	def := g.m.Tile(g.ground.At(col, row).Tile)
	return def == nil || def.Class == "Wall"
}

func (g *game) Update() error {
	g.anim.Update(render.Tick())
	now := g.anim.Clock
	p := &g.player

	if p.moving {
		t := min(float64(now-p.stepStart)/float64(stepTime), 1)
		toX, toY := float64(p.col*g.m.TileWidth), float64(p.row*g.m.TileHeight)
		p.x, p.y = p.fromX+(toX-p.fromX)*t, p.fromY+(toY-p.fromY)*t
		if t == 1 {
			p.moving = false
			g.collect()
		}
	}

	if !p.moving {
		dc, dr := input()
		if dc < 0 {
			p.flip = tilemap.FlipH
		} else if dc > 0 {
			p.flip = 0
		}
		if (dc != 0 || dr != 0) && !g.blocked(p.col+dc, p.row+dr) {
			if now-p.stepStart > stepTime+render.Tick() {
				p.walkStart = now
			}
			p.col, p.row = p.col+dc, p.row+dr
			p.fromX, p.fromY = p.x, p.y
			p.stepStart, p.moving = now, true
		}
	}
	return nil
}

func input() (dc, dr int) {
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA):
		return -1, 0
	case ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD):
		return 1, 0
	case ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW):
		return 0, -1
	case ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS):
		return 0, 1
	}
	return 0, 0
}

func (g *game) collect() {
	for _, c := range g.coins {
		if col, row := g.cellOf(c); c.Visible && col == g.player.col && row == g.player.row {
			c.Visible = false
			g.collected++
		}
	}
}

func (g *game) Draw(screen *ebiten.Image) {
	g.renderer.Draw(screen, g.m, render.Camera{})

	msg := fmt.Sprintf("coins %d/%d", g.collected, len(g.coins))
	if g.collected == len(g.coins) {
		msg += "  - all collected!"
	}
	ebitenutil.DebugPrint(screen, msg)
}

// afterLayer draws the player on top of the object layer.
func (g *game) afterLayer(dst *ebiten.Image, l tilemap.Layer, cam render.Camera) {
	if l != g.objects {
		return
	}
	p := &g.player

	// Walking plays the animation from its first frame; standing still shows frame one.
	var t time.Duration
	if p.moving {
		t = g.anim.Clock - p.walkStart
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(p.x-cam.X, p.y-cam.Y)
	g.renderer.DrawTile(dst, g.m, tilemap.Cell{Tile: p.tile, Flip: p.flip}, t, op)
}

func (g *game) Layout(int, int) (int, int) {
	return g.m.Width * g.m.TileWidth, g.m.Height * g.m.TileHeight
}
