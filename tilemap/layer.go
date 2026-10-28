package tilemap

import "iter"

type Layer interface {
	layer()
	Base() *LayerBase
}

type LayerBase struct {
	Name             string
	Class            string
	Visible          bool
	Opacity          float64
	OffsetX, OffsetY float64
	Props            Properties
}

func (b *LayerBase) Base() *LayerBase { return b }

type TileLayer struct {
	LayerBase
	Width, Height int
	Cells         []Cell
}

// At returns the cell at p, empty outside the layer.
func (l *TileLayer) At(p TilePos) Cell {
	if i := l.Index(p); i >= 0 {
		return l.Cells[i]
	}
	return Cell{}
}

// Index is p's index in Cells, -1 outside the layer.
func (l *TileLayer) Index(p TilePos) int {
	if p.Col < 0 || p.Row < 0 || p.Col >= l.Width || p.Row >= l.Height {
		return -1
	}
	return p.Row*l.Width + p.Col
}

// Pos is the position of Cells[i].
func (l *TileLayer) Pos(i int) TilePos {
	return TilePos{i % l.Width, i / l.Width}
}

// All yields every non-empty cell with its position, row by row.
func (l *TileLayer) All() iter.Seq2[TilePos, Cell] {
	return func(yield func(TilePos, Cell) bool) {
		for i, c := range l.Cells {
			if !c.Empty() && !yield(l.Pos(i), c) {
				return
			}
		}
	}
}

type ObjectLayer struct {
	LayerBase
	DrawOrder DrawOrder
	Objects   []MapObject
}

type DrawOrder uint8

const (
	DrawOrderTopDown DrawOrder = iota // by SortY, file order on ties
	DrawOrderIndex                    // file order
)

type ImageLayer struct {
	LayerBase
	Image ImageRef
}

type GroupLayer struct {
	LayerBase
	Layers []Layer
}

func (*TileLayer) layer()   {}
func (*ObjectLayer) layer() {}
func (*ImageLayer) layer()  {}
func (*GroupLayer) layer()  {}
