package tilemap

import "math"

// Vector is a 2D position, size or offset in pixels.
type Vector struct{ X, Y float64 }

func (v Vector) Add(o Vector) Vector { return Vector{v.X + o.X, v.Y + o.Y} }

func (v Vector) Sub(o Vector) Vector { return Vector{v.X - o.X, v.Y - o.Y} }

func (v Vector) Scale(f float64) Vector { return Vector{v.X * f, v.Y * f} }

// Rotate turns v around the origin by degrees, clockwise on screen.
func (v Vector) Rotate(degrees float64) Vector {
	if degrees == 0 {
		return v
	}
	sin, cos := math.Sincos(degrees * math.Pi / 180)
	return Vector{v.X*cos - v.Y*sin, v.X*sin + v.Y*cos}
}
