package tiled

import (
	"image/color"
	"path"
	"strconv"

	gotiled "github.com/lafriks/go-tiled"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

// convertProps resolves file properties against dir, the directory of the file that holds them.
func convertProps(ps gotiled.Properties, dir string) tilemap.Properties {
	if len(ps) == 0 {
		return nil
	}
	out := make(tilemap.Properties, len(ps))
	for _, p := range ps {
		out[p.Name] = propValue(p, dir)
	}
	return out
}

func propValue(p *gotiled.Property, dir string) any {
	switch p.Type {
	case "int":
		v, _ := strconv.Atoi(p.Value)
		return v
	case "float":
		v, _ := strconv.ParseFloat(p.Value, 64)
		return v
	case "bool":
		return p.Value == "true"
	case "color":
		if p.Value == "" {
			return color.RGBA{}
		}
		c, err := gotiled.ParseHexColor(p.Value)
		if err != nil {
			return color.RGBA{}
		}
		r, g, b, a := c.RGBA()
		return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	case "file":
		if p.Value == "" || path.IsAbs(p.Value) {
			return tilemap.FilePath(p.Value)
		}
		return tilemap.FilePath(cleanPath(path.Join(cleanPath(dir), p.Value)))
	case "object":
		v, _ := strconv.Atoi(p.Value)
		return tilemap.ObjectRef(v)
	case "class":
		if out := convertProps(p.Properties, dir); out != nil {
			return out
		}
		return tilemap.Properties{}
	}
	return p.Value
}
