package tilemap

import "math"

// MapObject is a spawn descriptor. Pos is the top-left corner; Rotation is in degrees, clockwise,
// around that corner.
type MapObject struct {
	ID         int
	Name, Type string
	Pos, Size  Vector
	Rotation   float64
	Visible    bool
	Opacity    float64
	Cell       Cell
	Shape      Shape
	Props      Properties
}

func (o *MapObject) IsTile() bool { return !o.Cell.Empty() }

// ToWorld returns the point at offset local from the top-left corner, rotated with the object.
func (o *MapObject) ToWorld(local Vector) Vector { return o.Pos.Add(local.Rotate(o.Rotation)) }

func (o *MapObject) Center() Vector { return o.ToWorld(o.Size.Scale(0.5)) }

// Foot is the bottom-center point, where a character stands.
func (o *MapObject) Foot() Vector { return o.ToWorld(Vector{o.Size.X / 2, o.Size.Y}) }

// Bounds is the axis-aligned box around the rotated object.
func (o *MapObject) Bounds() (minimum, maximum Vector) {
	pts := o.Shape.Points
	if len(pts) == 0 || (o.Shape.Kind != ShapePolygon && o.Shape.Kind != ShapePolyline) {
		pts = []Vector{{0, 0}, {o.Size.X, 0}, {0, o.Size.Y}, o.Size}
	}

	minimum = Vector{math.Inf(1), math.Inf(1)}
	maximum = Vector{math.Inf(-1), math.Inf(-1)}
	for _, p := range pts {
		q := o.ToWorld(p)
		minimum = Vector{min(minimum.X, q.X), min(minimum.Y, q.Y)}
		maximum = Vector{max(maximum.X, q.X), max(maximum.Y, q.Y)}
	}
	return minimum, maximum
}

// SortY is the bottom edge, for depth sorting.
func (o *MapObject) SortY() float64 {
	_, maximum := o.Bounds()
	return maximum.Y
}

type ShapeKind uint8

const (
	ShapeRect ShapeKind = iota
	ShapeEllipse
	ShapePolygon
	ShapePolyline
	ShapePoint
)

// Shape is relative to its owner; Points are relative to Pos.
type Shape struct {
	Kind      ShapeKind
	Pos, Size Vector
	Rotation  float64
	Points    []Vector
}
