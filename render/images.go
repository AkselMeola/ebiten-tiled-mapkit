package render

import (
	"errors"
	"fmt"
	"image"
	"maps"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// Images is a set of named images outside the map, such as UI graphics.
type Images struct {
	textures TextureProvider
	entries  map[string]imageEntry
}

type imageEntry struct {
	path   tilemap.ImageRef
	region image.Rectangle // empty for the whole image
	img    *ebiten.Image
}

// NewImages loads through textures; pass the renderer's to share its cache.
func NewImages(textures TextureProvider) *Images {
	return &Images{textures: textures, entries: make(map[string]imageEntry)}
}

// Add registers the image at path under name, replacing an earlier one.
func (i *Images) Add(name, path string) *Images {
	return i.AddRegion(name, path, image.Rectangle{})
}

// AddRegion registers part of the image at path, such as one control on a UI sheet.
func (i *Images) AddRegion(name, path string, region image.Rectangle) *Images {
	i.entries[name] = imageEntry{path: tilemap.ImageRef(path), region: region}
	return i
}

// Load loads every added image and returns all errors.
func (i *Images) Load() error {
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(i.entries)) {
		if _, err := i.load(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Get returns the image added under name, loading it on first use. It panics on an unknown name
// or a failed load; call Load first to handle errors.
func (i *Images) Get(name string) *ebiten.Image {
	img, err := i.load(name)
	if err != nil {
		panic(err)
	}
	return img
}

func (i *Images) load(name string) (*ebiten.Image, error) {
	e, ok := i.entries[name]
	if !ok {
		return nil, fmt.Errorf("render: image %q not added", name)
	}
	if e.img != nil {
		return e.img, nil
	}
	img, err := i.textures.Texture(e.path)
	if err != nil {
		return nil, err
	}
	if !e.region.Empty() {
		if !e.region.In(img.Bounds()) {
			return nil, fmt.Errorf("render: image %q: region %v outside %q %v", name, e.region, e.path, img.Bounds())
		}
		img = img.SubImage(e.region).(*ebiten.Image)
	}
	e.img = img
	i.entries[name] = e
	return img, nil
}
