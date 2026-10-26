package render

import (
	"fmt"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

type TextureProvider interface {
	Texture(ref tilemap.ImageRef) (*ebiten.Image, error)
}

type FSTextures struct {
	fsys  fs.FS
	cache map[tilemap.ImageRef]textureEntry
}

type textureEntry struct {
	img *ebiten.Image
	err error
}

func NewFSTextures(fsys fs.FS) *FSTextures {
	return &FSTextures{fsys: fsys, cache: make(map[tilemap.ImageRef]textureEntry)}
}

func (t *FSTextures) Texture(ref tilemap.ImageRef) (*ebiten.Image, error) {
	if e, ok := t.cache[ref]; ok {
		return e.img, e.err
	}
	img, _, err := ebitenutil.NewImageFromFileSystem(t.fsys, string(ref))
	if err != nil {
		err = fmt.Errorf("render: texture %q: %w", ref, err)
	}
	t.cache[ref] = textureEntry{img: img, err: err}
	return img, err
}
