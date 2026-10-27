package tiled

import (
	"fmt"
	"maps"

	gotiled "github.com/lafriks/go-tiled"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

const (
	flipHMask    = 0x80000000
	flipVMask    = 0x40000000
	flipDMask    = 0x20000000
	flipHex120   = 0x10000000
	flipAllMasks = flipHMask | flipVMask | flipDMask | flipHex120
)

func (c *converter) objectLayer(g *gotiled.ObjectGroup) (*tilemap.ObjectLayer, error) {
	out := &tilemap.ObjectLayer{
		LayerBase: layerBase(g.Name, g.Class, g.Visible, g.Opacity, g.OffsetX, g.OffsetY, g.Properties),
		Objects:   make([]tilemap.MapObject, 0, len(g.Objects)),
	}
	if g.DrawOrder == "index" {
		out.DrawOrder = tilemap.DrawOrderIndex
	}
	for _, o := range g.Objects {
		obj, err := c.object(o)
		if err != nil {
			return nil, fmt.Errorf("object layer %q: object %d: %w", g.Name, o.ID, err)
		}
		out.Objects = append(out.Objects, obj)
	}
	return out, nil
}

func (c *converter) object(o *gotiled.Object) (tilemap.MapObject, error) {
	src := *o
	props := convertProps(o.Properties)

	if o.Template != nil && o.Template.Object != nil {
		t := o.Template.Object
		if src.Name == "" {
			src.Name = t.Name
		}
		if src.Type == "" {
			src.Type = t.Type
		}
		if src.Width == 0 && src.Height == 0 {
			src.Width, src.Height = t.Width, t.Height
		}
		if src.Rotation == 0 {
			src.Rotation = t.Rotation
		}
		if src.Opacity == 1 {
			src.Opacity = t.Opacity
		}
		if len(src.Ellipses)+len(src.Polygons)+len(src.PolyLines) == 0 {
			src.Ellipses, src.Polygons, src.PolyLines = t.Ellipses, t.Polygons, t.PolyLines
		}
		if src.GID == 0 && t.GID != 0 {
			gid, err := c.templateGID(o.Template)
			if err != nil {
				return tilemap.MapObject{}, err
			}
			src.GID = gid
		}
		merged := convertProps(t.Properties)
		if merged == nil {
			merged = props
		} else {
			maps.Copy(merged, props)
		}
		props = merged
	}

	out := tilemap.MapObject{
		ID:       int(src.ID),
		Name:     src.Name,
		Type:     src.Type,
		Pos:      tilemap.Vector{X: src.X, Y: src.Y},
		Size:     tilemap.Vector{X: src.Width, Y: src.Height},
		Rotation: src.Rotation,
		Visible:  src.Visible,
		Opacity:  float64(src.Opacity),
		Props:    props,
	}

	if src.GID != 0 {
		cell, err := c.cell(src.GID)
		if err != nil {
			return tilemap.MapObject{}, err
		}
		out.Tile = cell
		if def := c.m.TileDef(cell.TileID); def != nil {
			if out.Size == (tilemap.Vector{}) {
				out.Size = tilemap.Vector{X: float64(def.Src.Dx()), Y: float64(def.Src.Dy())}
			}
			// As in Tiled, a tile object inherits its tile's class and properties.
			if out.Type == "" {
				out.Type = def.Class
			}
			if inherited := maps.Clone(def.Props); inherited != nil {
				maps.Copy(inherited, out.Props)
				out.Props = inherited
			}
		}
		anchor := c.objectAnchor(src.GID)
		offset := tilemap.Vector{X: anchor.X * out.Size.X, Y: anchor.Y * out.Size.Y}
		out.Pos = out.Pos.Sub(offset.Rotate(out.Rotation))
		out.Shape = tilemap.Shape{Kind: tilemap.ShapeRect, Size: out.Size}
		return out, nil
	}

	out.Shape = shape(&src)
	return out, nil
}

func (c *converter) cell(gid uint32) (tilemap.Cell, error) {
	id, ok := c.gidToTile[gid&^flipAllMasks]
	if !ok {
		return tilemap.Cell{}, fmt.Errorf("unknown tile gid %d", gid&^flipAllMasks)
	}
	var flip tilemap.Flip
	if gid&flipHMask != 0 {
		flip |= tilemap.FlipH
	}
	if gid&flipVMask != 0 {
		flip |= tilemap.FlipV
	}
	if gid&flipDMask != 0 {
		flip |= tilemap.FlipD
	}
	return tilemap.Cell{TileID: id, Flip: flip}, nil
}

// objectAnchor is where Tiled anchors a tile object, as fractions of its size.
func (c *converter) objectAnchor(gid uint32) tilemap.Vector {
	bare := gid &^ flipAllMasks
	alignment := "unspecified"
	for i := len(c.tm.Tilesets) - 1; i >= 0; i-- {
		if c.tm.Tilesets[i].FirstGID <= bare {
			alignment = c.tm.Tilesets[i].ObjectAlignment
			break
		}
	}

	switch alignment {
	case "topleft":
		return tilemap.Vector{X: 0, Y: 0}
	case "top":
		return tilemap.Vector{X: 0.5, Y: 0}
	case "topright":
		return tilemap.Vector{X: 1, Y: 0}
	case "left":
		return tilemap.Vector{X: 0, Y: 0.5}
	case "center":
		return tilemap.Vector{X: 0.5, Y: 0.5}
	case "right":
		return tilemap.Vector{X: 1, Y: 0.5}
	case "bottom":
		return tilemap.Vector{X: 0.5, Y: 1}
	case "bottomright":
		return tilemap.Vector{X: 1, Y: 1}
	case "bottomleft":
		return tilemap.Vector{X: 0, Y: 1}
	}
	if c.m.Orientation == tilemap.OrientationIsometric {
		return tilemap.Vector{X: 0.5, Y: 1}
	}
	return tilemap.Vector{X: 0, Y: 1}
}

// templateGID maps a template's tileset-relative GID onto the map's tileset with the same source.
func (c *converter) templateGID(t *gotiled.Template) (uint32, error) {
	gid := t.Object.GID
	if t.Tileset == nil || t.Tileset.Source == "" {
		return 0, fmt.Errorf("template tile object without external tileset")
	}
	src := cleanPath(c.tm.GetFileFullPath(t.Tileset.Source))
	for _, ts := range c.tm.Tilesets {
		if ts.Source != "" && cleanPath(c.tm.GetFileFullPath(ts.Source)) == src {
			return (gid & flipAllMasks) | (ts.FirstGID + (gid&^flipAllMasks - t.Tileset.FirstGID)), nil
		}
	}
	return 0, fmt.Errorf("template tileset %q is not used by the map", t.Tileset.Source)
}

func shape(o *gotiled.Object) tilemap.Shape {
	s := tilemap.Shape{Kind: tilemap.ShapeRect, Size: tilemap.Vector{X: o.Width, Y: o.Height}}
	switch {
	case len(o.Polygons) > 0:
		s.Kind = tilemap.ShapePolygon
		s.Points = points(o.Polygons[0].Points)
	case len(o.PolyLines) > 0:
		s.Kind = tilemap.ShapePolyline
		s.Points = points(o.PolyLines[0].Points)
	case len(o.Ellipses) > 0:
		s.Kind = tilemap.ShapeEllipse
	case o.Width == 0 && o.Height == 0 && o.Text == nil:
		// go-tiled does not parse <point/>; a sizeless plain object is the closest match.
		s.Kind = tilemap.ShapePoint
	}
	return s
}

func points(ps *gotiled.Points) []tilemap.Vector {
	if ps == nil {
		return nil
	}
	out := make([]tilemap.Vector, len(*ps))
	for i, p := range *ps {
		out[i] = tilemap.Vector{X: p.X, Y: p.Y}
	}
	return out
}
