package tilemap

// TilePos is a position on the tile grid, in tiles. Pixel positions are Vectors.
type TilePos struct{ Col, Row int }

func (p TilePos) Add(o TilePos) TilePos { return TilePos{p.Col + o.Col, p.Row + o.Row} }

// Steps to the neighboring tiles, for pos.Add(Right).
var (
	Up    = TilePos{0, -1}
	Down  = TilePos{0, 1}
	Left  = TilePos{-1, 0}
	Right = TilePos{1, 0}
)
