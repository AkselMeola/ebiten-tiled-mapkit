package render

import (
	"flag"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var frames chan func()

// TestMain runs benchmarks next to Ebiten's game loop; inFrame executes code inside one frame,
// so drawing is flushed per frame as in a game. Plain tests stay headless.
func TestMain(m *testing.M) {
	flag.Parse()
	if flag.Lookup("test.bench").Value.String() == "" {
		os.Exit(m.Run())
	}

	frames = make(chan func())
	g := &testGame{exit: make(chan int, 1)}
	go func() { g.exit <- m.Run() }()

	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetWindowSize(320, 240)
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}

func inFrame(f func()) {
	done := make(chan struct{})
	frames <- func() { f(); close(done) }
	<-done
}

type testGame struct {
	exit chan int
	code int
}

func (g *testGame) Update() error {
	select {
	case f := <-frames:
		f()
	case g.code = <-g.exit:
		return ebiten.Termination
	}
	return nil
}

func (g *testGame) Draw(*ebiten.Image) {}

func (g *testGame) Layout(int, int) (int, int) { return 320, 240 }
