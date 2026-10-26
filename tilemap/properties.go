package tilemap

import "image/color"

// Properties values are string, int, float64, bool, color.RGBA, FilePath, ObjectRef or Properties.
type Properties map[string]any

type FilePath string

type ObjectRef int

func (p Properties) String(name string) string {
	switch v := p[name].(type) {
	case string:
		return v
	case FilePath:
		return string(v)
	}
	return ""
}

func (p Properties) Int(name string) int {
	switch v := p[name].(type) {
	case int:
		return v
	case ObjectRef:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func (p Properties) Float(name string) float64 {
	switch v := p[name].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func (p Properties) Bool(name string) bool {
	v, _ := p[name].(bool)
	return v
}

func (p Properties) Color(name string) (color.RGBA, bool) {
	v, ok := p[name].(color.RGBA)
	return v, ok
}

func (p Properties) Class(name string) Properties {
	v, _ := p[name].(Properties)
	return v
}
