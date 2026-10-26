package tilemap

import "math"

// MapObject is a spawn descriptor. X, Y is the top-left corner; Rotation is in degrees, clockwise,
// around that corner.
type MapObject struct {
	ID         int
	Name, Type string
	X, Y, W, H float64
	Rotation   float64
	Visible    bool
	Opacity    float64
	Tile       Cell
	Shape      Shape
	Props      Properties
}

func (o *MapObject) IsTile() bool { return !o.Tile.Empty() }

// Bounds is the axis-aligned box around the rotated object.
func (o *MapObject) Bounds() (minX, minY, maxX, maxY float64) {
	pts := o.Shape.Points
	if len(pts) == 0 || (o.Shape.Kind != ShapePolygon && o.Shape.Kind != ShapePolyline) {
		pts = []Point{{0, 0}, {o.W, 0}, {0, o.H}, {o.W, o.H}}
	}

	sin, cos := math.Sincos(o.Rotation * math.Pi / 180)
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		x := o.X + p.X*cos - p.Y*sin
		y := o.Y + p.X*sin + p.Y*cos
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	return minX, minY, maxX, maxY
}

// SortY is the bottom edge, for depth sorting.
func (o *MapObject) SortY() float64 {
	_, _, _, maxY := o.Bounds()
	return maxY
}

type ShapeKind uint8

const (
	ShapeRect ShapeKind = iota
	ShapeEllipse
	ShapePolygon
	ShapePolyline
	ShapePoint
)

type Point struct{ X, Y float64 }

// Shape is relative to its owner; Points are relative to X, Y.
type Shape struct {
	Kind     ShapeKind
	X, Y     float64
	W, H     float64
	Rotation float64
	Points   []Point
}
