package tiled

import (
	"image/color"
	"strconv"

	gotiled "github.com/lafriks/go-tiled"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

func convertProps(ps gotiled.Properties) tilemap.Properties {
	if len(ps) == 0 {
		return nil
	}
	out := make(tilemap.Properties, len(ps))
	for _, p := range ps {
		out[p.Name] = propValue(p)
	}
	return out
}

func propValue(p *gotiled.Property) any {
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
		return tilemap.FilePath(p.Value)
	case "object":
		v, _ := strconv.Atoi(p.Value)
		return tilemap.ObjectRef(v)
	case "class":
		if out := convertProps(p.Properties); out != nil {
			return out
		}
		return tilemap.Properties{}
	}
	return p.Value
}
