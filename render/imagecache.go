package render

import (
	"fmt"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// ImageSource loads images by path. The renderer and Images load through it.
type ImageSource interface {
	Image(ref tilemap.ImageRef) (*ebiten.Image, error)
}

// ImageCache loads images from a file system and keeps them, errors included. Create one for
// the whole game.
type ImageCache struct {
	fsys  fs.FS
	cache map[tilemap.ImageRef]cacheEntry
}

type cacheEntry struct {
	img *ebiten.Image
	err error
}

func NewImageCache(fsys fs.FS) *ImageCache {
	return &ImageCache{fsys: fsys, cache: make(map[tilemap.ImageRef]cacheEntry)}
}

func (c *ImageCache) Image(ref tilemap.ImageRef) (*ebiten.Image, error) {
	if e, ok := c.cache[ref]; ok {
		return e.img, e.err
	}
	img, _, err := ebitenutil.NewImageFromFileSystem(c.fsys, string(ref))
	if err != nil {
		err = fmt.Errorf("render: image %q: %w", ref, err)
	}
	c.cache[ref] = cacheEntry{img: img, err: err}
	return img, err
}
