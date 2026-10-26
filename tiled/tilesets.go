package tiled

import (
	"fmt"
	"image"
	"time"

	gotiled "github.com/lafriks/go-tiled"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

type pendingAnimation struct {
	tile     tilemap.TileID
	firstGID uint32
	frames   []*gotiled.AnimationFrame
}

func (c *converter) loadTilesets() error {
	var anims []pendingAnimation

	for i, ts := range c.tm.Tilesets {
		// go-tiled initializes tilesets lazily, only when a GID references them.
		if !ts.SourceLoaded && ts.Source == "" {
			ts.SetBaseDir(c.tm.GetFileFullPath(""))
			ts.SourceLoaded = true
		}
		if !ts.SourceLoaded {
			loaded, err := gotiled.LoadTilesetFile(cleanPath(c.tm.GetFileFullPath(ts.Source)), gotiled.WithFileSystem(c.fsys))
			if err != nil {
				return fmt.Errorf("tileset %q: %w", ts.Source, err)
			}
			loaded.FirstGID = ts.FirstGID
			loaded.Source = ts.Source
			loaded.SourceLoaded = true
			c.tm.Tilesets[i] = loaded
			ts = loaded
		}

		a, err := c.addTileset(ts)
		if err != nil {
			return fmt.Errorf("tileset %q: %w", ts.Name, err)
		}
		anims = append(anims, a...)
	}

	for _, a := range anims {
		frames := make([]tilemap.Frame, len(a.frames))
		for i, f := range a.frames {
			id, ok := c.gidToTile[a.firstGID+f.TileID]
			if !ok {
				return fmt.Errorf("animation of tile %d references unknown tile %d", a.tile, f.TileID)
			}
			frames[i] = tilemap.Frame{Tile: id, Duration: time.Duration(f.Duration) * time.Millisecond}
		}
		c.m.Tiles[a.tile].SetAnimation(frames)
	}

	return nil
}

func (c *converter) addTileset(ts *gotiled.Tileset) ([]pendingAnimation, error) {
	var offset image.Point
	if ts.TileOffset != nil {
		offset = image.Pt(ts.TileOffset.X, ts.TileOffset.Y)
	}

	meta := make(map[uint32]*gotiled.TilesetTile, len(ts.Tiles))
	for _, t := range ts.Tiles {
		meta[t.ID] = t
	}

	var anims []pendingAnimation
	add := func(localID uint32, def tilemap.TileDef) {
		def.Offset = offset
		if t := meta[localID]; t != nil {
			def.Class = t.Class
			def.Props = convertProps(t.Properties)
			def.Colliders = colliders(t.ObjectGroups)
		}

		id := tilemap.TileID(len(c.m.Tiles))
		c.m.Tiles = append(c.m.Tiles, def)
		c.gidToTile[ts.FirstGID+localID] = id

		if t := meta[localID]; t != nil && len(t.Animation) > 0 {
			anims = append(anims, pendingAnimation{tile: id, firstGID: ts.FirstGID, frames: t.Animation})
		}
	}

	if ts.Image != nil {
		if ts.TileWidth <= 0 || ts.TileHeight <= 0 {
			return nil, fmt.Errorf("invalid tile size %dx%d", ts.TileWidth, ts.TileHeight)
		}
		img := tilemap.ImageRef(cleanPath(ts.GetFileFullPath(ts.Image.Source)))
		for i := range atlasTileCount(ts) {
			add(uint32(i), tilemap.TileDef{Image: img, Src: ts.GetTileRect(uint32(i))})
		}
		return anims, nil
	}

	for _, t := range ts.Tiles {
		if t.Image == nil {
			continue
		}
		add(t.ID, tilemap.TileDef{
			Image: tilemap.ImageRef(cleanPath(ts.GetFileFullPath(t.Image.Source))),
			Src:   image.Rect(t.X, t.Y, t.X+t.Width, t.Y+t.Height),
		})
	}
	return anims, nil
}

func atlasTileCount(ts *gotiled.Tileset) int {
	if ts.TileCount > 0 {
		return ts.TileCount
	}
	cols := ts.Columns
	if cols == 0 {
		cols = (ts.Image.Width - 2*ts.Margin + ts.Spacing) / (ts.TileWidth + ts.Spacing)
	}
	rows := (ts.Image.Height - 2*ts.Margin + ts.Spacing) / (ts.TileHeight + ts.Spacing)
	return max(cols*rows, 0)
}

func colliders(groups []*gotiled.ObjectGroup) []tilemap.Shape {
	var shapes []tilemap.Shape
	for _, g := range groups {
		for _, o := range g.Objects {
			s := shape(o)
			s.X, s.Y, s.Rotation = o.X, o.Y, o.Rotation
			shapes = append(shapes, s)
		}
	}
	return shapes
}
