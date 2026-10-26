package render

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Animator struct {
	Clock time.Duration
}

// Update must be called from ebiten's Update, never from Draw.
func (a *Animator) Update(dt time.Duration) {
	a.Clock += dt
}

// Tick is the fixed duration of one ebiten update.
func Tick() time.Duration {
	return time.Second / time.Duration(ebiten.TPS())
}
