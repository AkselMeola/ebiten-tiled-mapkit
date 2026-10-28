// Package tiled loads Tiled .tmx maps into tilemap.Map.
package tiled

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	gotiled "github.com/lafriks/go-tiled"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

var (
	ErrUnsupportedFormat = errors.New("tiled: unsupported map format")
	ErrInfiniteMap       = errors.New("tiled: infinite maps are not supported")
)

// Load reads the map at name from fsys. Tilesets and images are resolved relative to it within fsys.
func Load(fsys fs.FS, name string) (*tilemap.Map, error) {
	if ext := strings.ToLower(path.Ext(name)); ext != ".tmx" {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedFormat, ext)
	}

	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("tiled: read %q: %w", name, err)
	}

	lay, err := scanLayout(data)
	if err != nil {
		return nil, fmt.Errorf("tiled: scan %q: %w", name, err)
	}
	if lay.infinite {
		return nil, fmt.Errorf("%w: %q", ErrInfiniteMap, name)
	}

	tm, err := gotiled.LoadReader(path.Dir(name), bytes.NewReader(data), gotiled.WithFileSystem(fsys))
	if err != nil {
		return nil, fmt.Errorf("tiled: parse %q: %w", name, err)
	}

	m := &tilemap.Map{
		Width:       tm.Width,
		Height:      tm.Height,
		TileWidth:   tm.TileWidth,
		TileHeight:  tm.TileHeight,
		Orientation: orientation(tm.Orientation),
		Tiles:       []tilemap.TileDef{{}},
	}
	if tm.Properties != nil {
		m.Props = convertProps(*tm.Properties, tm.GetFileFullPath(""))
	}

	c := &converter{
		fsys:      fsys,
		tm:        tm,
		m:         m,
		gidToTile: make(map[uint32]tilemap.TileID),
	}

	if err := c.loadTilesets(); err != nil {
		return nil, fmt.Errorf("tiled: %q: %w", name, err)
	}

	m.Layers, err = c.layers(lay.layers, tm.Layers, tm.ObjectGroups, tm.ImageLayers, tm.Groups)
	if err != nil {
		return nil, fmt.Errorf("tiled: %q: %w", name, err)
	}

	return m, nil
}

type converter struct {
	fsys      fs.FS
	tm        *gotiled.Map
	m         *tilemap.Map
	gidToTile map[uint32]tilemap.TileID
}

func orientation(s string) tilemap.Orientation {
	switch s {
	case "orthogonal":
		return tilemap.OrientationOrthogonal
	case "isometric":
		return tilemap.OrientationIsometric
	case "staggered":
		return tilemap.OrientationStaggered
	case "hexagonal":
		return tilemap.OrientationHexagonal
	}
	return tilemap.OrientationUnknown
}

func cleanPath(p string) string {
	return path.Clean(filepath.ToSlash(p))
}
