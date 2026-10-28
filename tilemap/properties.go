package tilemap

import "image/color"

// Properties values are string, int, float64, bool, color.RGBA, FilePath, ObjectRef or Properties.
// Getters return the zero value for a missing or differently typed property.
type Properties map[string]any

// FilePath is a slash-separated path within the file system the map was loaded from.
type FilePath string

// ObjectRef is a MapObject.ID; look it up with Map.Object.
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

func (p Properties) Color(name string) color.RGBA {
	v, _ := p[name].(color.RGBA)
	return v
}

// Nested returns a class property's members.
func (p Properties) Nested(name string) Properties {
	v, _ := p[name].(Properties)
	return v
}
