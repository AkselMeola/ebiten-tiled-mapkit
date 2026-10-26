package tiled

import (
	"errors"
	"fmt"

	gotiled "github.com/lafriks/go-tiled"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

var errLayoutMismatch = errors.New("layer layout does not match parsed layers")

func (c *converter) layers(
	nodes []layoutNode,
	tileLayers []*gotiled.Layer,
	objectGroups []*gotiled.ObjectGroup,
	imageLayers []*gotiled.ImageLayer,
	groups []*gotiled.Group,
) ([]tilemap.Layer, error) {
	var ti, oi, ii, gi int
	out := make([]tilemap.Layer, 0, len(nodes))

	for _, n := range nodes {
		switch n.kind {
		case "layer":
			if ti >= len(tileLayers) {
				return nil, errLayoutMismatch
			}
			l, err := c.tileLayer(tileLayers[ti])
			if err != nil {
				return nil, err
			}
			out = append(out, l)
			ti++
		case "objectgroup":
			if oi >= len(objectGroups) {
				return nil, errLayoutMismatch
			}
			l, err := c.objectLayer(objectGroups[oi])
			if err != nil {
				return nil, err
			}
			out = append(out, l)
			oi++
		case "imagelayer":
			if ii >= len(imageLayers) {
				return nil, errLayoutMismatch
			}
			out = append(out, c.imageLayer(imageLayers[ii]))
			ii++
		case "group":
			if gi >= len(groups) {
				return nil, errLayoutMismatch
			}
			g := groups[gi]
			children, err := c.layers(n.children, g.Layers, g.ObjectGroups, g.ImageLayers, g.Groups)
			if err != nil {
				return nil, err
			}
			out = append(out, &tilemap.GroupLayer{
				LayerBase: layerBase(g.Name, g.Class, g.Visible, g.Opacity, g.OffsetX, g.OffsetY, g.Properties),
				Layers:    children,
			})
			gi++
		}
	}

	if ti != len(tileLayers) || oi != len(objectGroups) || ii != len(imageLayers) || gi != len(groups) {
		return nil, errLayoutMismatch
	}
	return out, nil
}

func layerBase(name, class string, visible bool, opacity float32, offX, offY float64, props gotiled.Properties) tilemap.LayerBase {
	return tilemap.LayerBase{
		Name:    name,
		Class:   class,
		Visible: visible,
		Opacity: float64(opacity),
		OffsetX: offX,
		OffsetY: offY,
		Props:   convertProps(props),
	}
}

func (c *converter) tileLayer(l *gotiled.Layer) (*tilemap.TileLayer, error) {
	out := &tilemap.TileLayer{
		LayerBase: layerBase(l.Name, l.Class, l.Visible, l.Opacity, l.OffsetX, l.OffsetY, l.Properties),
		W:         c.tm.Width,
		H:         c.tm.Height,
		Cells:     make([]tilemap.Cell, len(l.Tiles)),
	}

	for i, t := range l.Tiles {
		if t.IsNil() {
			continue
		}
		gid := t.Tileset.FirstGID + t.ID
		id, ok := c.gidToTile[gid]
		if !ok {
			return nil, fmt.Errorf("layer %q: unknown tile gid %d", l.Name, gid)
		}

		var flip tilemap.Flip
		if t.HorizontalFlip {
			flip |= tilemap.FlipH
		}
		if t.VerticalFlip {
			flip |= tilemap.FlipV
		}
		if t.DiagonalFlip {
			flip |= tilemap.FlipD
		}
		out.Cells[i] = tilemap.Cell{Tile: id, Flip: flip}
	}

	return out, nil
}

func (c *converter) imageLayer(l *gotiled.ImageLayer) *tilemap.ImageLayer {
	out := &tilemap.ImageLayer{
		LayerBase: layerBase(l.Name, l.Class, l.Visible, l.Opacity, l.OffsetX+float64(l.X), l.OffsetY+float64(l.Y), l.Properties),
	}
	if l.Image != nil && l.Image.Source != "" {
		out.Image = tilemap.ImageRef(cleanPath(c.tm.GetFileFullPath(l.Image.Source)))
	}
	return out
}
