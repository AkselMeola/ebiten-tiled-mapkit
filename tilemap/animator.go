package tilemap

import "time"

// Animator plays one tile's animation for one game entity: looping, or a number of times before
// switching to another tile. Advance it from the game's Update and draw Frame.
type Animator struct {
	m       *Map
	tile    TileID
	elapsed time.Duration
	times   int // full cycles to play before switching to then; 0 loops forever
	then    TileID
	done    bool
}

func NewAnimator(m *Map) *Animator {
	return &Animator{m: m}
}

// Play loops id. Playing the tile that is already looping doesn't restart it.
func (a *Animator) Play(id TileID) {
	if id == a.tile && a.times == 0 {
		return
	}
	a.start(id, 0, 0)
}

// PlayTimes plays id n times from its first frame, then loops then. n <= 0 loops id forever.
func (a *Animator) PlayTimes(id TileID, n int, then TileID) {
	a.start(id, max(n, 0), then)
}

// PlayOnce plays id once from its first frame, then loops then.
func (a *Animator) PlayOnce(id, then TileID) {
	a.PlayTimes(id, 1, then)
}

func (a *Animator) start(id TileID, times int, then TileID) {
	a.tile, a.elapsed, a.times, a.then, a.done = id, 0, times, then, false
}

// Update advances the animation by dt, usually render.Tick().
func (a *Animator) Update(dt time.Duration) {
	a.done = false
	a.elapsed += dt
	if a.times == 0 {
		return
	}
	var length time.Duration
	if def := a.m.TileDef(a.tile); def != nil {
		length = def.AnimationDuration()
	}
	if a.elapsed >= time.Duration(a.times)*length {
		a.start(a.then, 0, 0)
		a.done = true
	}
}

// Done reports whether a PlayTimes or PlayOnce animation finished in the last Update. It is true
// for that one tick; the animator then already plays the then tile.
func (a *Animator) Done() bool { return a.done }

// Tile is the tile being played.
func (a *Animator) Tile() TileID { return a.tile }

// Frame is the tile to draw now: the current frame of Tile's animation, or Tile if it isn't
// animated.
func (a *Animator) Frame() TileID { return a.m.Frame(a.tile, a.elapsed) }
