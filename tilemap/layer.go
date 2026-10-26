package tilemap

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
	W, H  int
	Cells []Cell
}

func (l *TileLayer) At(col, row int) Cell {
	if col < 0 || row < 0 || col >= l.W || row >= l.H {
		return Cell{}
	}
	return l.Cells[row*l.W+col]
}

type ObjectLayer struct {
	LayerBase
	Objects []MapObject
}

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
