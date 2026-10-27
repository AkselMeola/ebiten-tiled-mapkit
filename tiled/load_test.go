package tiled

import (
	"errors"
	"image"
	"image/color"
	"io/fs"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/AkselMeola/ebiten-tiled-mapkit/tilemap"
)

const testTMX = `<?xml version="1.0" encoding="UTF-8"?>
<map version="1.10" orientation="orthogonal" renderorder="right-down" width="2" height="2" tilewidth="16" tileheight="16" infinite="0">
 <properties>
  <property name="level" type="int" value="3"/>
  <property name="tint" type="color" value="#80ff0000"/>
  <property name="spawn" type="class" propertytype="Spawn">
   <properties>
    <property name="count" type="int" value="2"/>
   </properties>
  </property>
 </properties>
 <tileset firstgid="1" name="atlas" tilewidth="16" tileheight="16" tilecount="4" columns="2">
  <image source="../img/tiles.png" width="32" height="32"/>
  <tile id="0">
   <animation>
    <frame tileid="0" duration="100"/>
    <frame tileid="1" duration="300"/>
   </animation>
  </tile>
  <tile id="1" type="Wall">
   <properties>
    <property name="solid" type="bool" value="true"/>
   </properties>
   <objectgroup draworder="index">
    <object id="1" x="2" y="3" width="10" height="11"/>
   </objectgroup>
  </tile>
 </tileset>
 <tileset firstgid="10" source="sets/objs.tsx"/>
 <tileset firstgid="20" source="sets/unused.tsx"/>
 <layer id="1" name="ground" width="2" height="2">
  <data encoding="csv">1,2,0,4</data>
 </layer>
 <objectgroup id="2" name="objs" offsetx="5" offsety="6">
  <object id="1" name="crate" type="Box" gid="2147483663" x="32" y="48" width="16" height="24" opacity="0.5"/>
  <object id="2" name="zone" x="1" y="2">
   <polygon points="0,0 4,0 4,4"/>
  </object>
  <object id="3" name="spawn" x="7" y="8">
   <point/>
  </object>
  <object id="4" x="0" y="0" width="5" height="6">
   <ellipse/>
  </object>
  <object id="5" template="../tpl/crate.tx" x="64" y="64">
   <properties>
    <property name="hp" type="int" value="9"/>
   </properties>
  </object>
 </objectgroup>
 <imagelayer id="3" name="bg" offsetx="1" offsety="2" opacity="0.5">
  <image source="../img/bg.png" width="10" height="10"/>
 </imagelayer>
 <group id="4" name="g" visible="0">
  <layer id="5" name="top" width="2" height="2">
   <data encoding="base64" compression="zlib">eJwFwQEBAAAIAiDM404PgmNQPATEAIc=</data>
  </layer>
 </group>
</map>
`

const testObjsTSX = `<?xml version="1.0" encoding="UTF-8"?>
<tileset name="objs" tilewidth="16" tileheight="24" tilecount="1" columns="0">
 <tileoffset x="1" y="-2"/>
 <tile id="5" type="Box">
  <image source="../../img/crate.png" width="16" height="24"/>
 </tile>
</tileset>
`

const testUnusedTSX = `<?xml version="1.0" encoding="UTF-8"?>
<tileset name="unused" tilewidth="8" tileheight="8" tilecount="2" columns="2">
 <image source="../../img/unused.png" width="16" height="8"/>
</tileset>
`

const testTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<template>
 <tileset firstgid="1" source="../maps/sets/objs.tsx"/>
 <object name="tpl-crate" type="Box" gid="6" width="16" height="24">
  <properties>
   <property name="hp" type="int" value="1"/>
   <property name="kind" value="wood"/>
  </properties>
 </object>
</template>
`

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"maps/test.tmx":        {Data: []byte(testTMX)},
		"maps/sets/objs.tsx":   {Data: []byte(testObjsTSX)},
		"maps/sets/unused.tsx": {Data: []byte(testUnusedTSX)},
		"tpl/crate.tx":         {Data: []byte(testTemplate)},
	}
}

func TestLoad(t *testing.T) {
	m, err := Load(testFS(), "maps/test.tmx")
	if err != nil {
		t.Fatal(err)
	}

	if m.Width != 2 || m.Height != 2 || m.TileWidth != 16 || m.TileHeight != 16 {
		t.Errorf("size = %dx%d @ %dx%d", m.Width, m.Height, m.TileWidth, m.TileHeight)
	}
	if m.Orientation != tilemap.OrientationOrthogonal {
		t.Errorf("orientation = %v", m.Orientation)
	}

	// sentinel + 4 atlas tiles + 1 collection tile + 2 unused tiles
	if len(m.Tiles) != 8 {
		t.Fatalf("len(Tiles) = %d, want 8", len(m.Tiles))
	}

	t.Run("properties", func(t *testing.T) {
		if got := m.Props.Int("level"); got != 3 {
			t.Errorf("level = %d", got)
		}
		if got, _ := m.Props.Color("tint"); got != (color.RGBA{R: 255, A: 128}) {
			t.Errorf("tint = %v", got)
		}
		if got := m.Props.Class("spawn").Int("count"); got != 2 {
			t.Errorf("spawn.count = %d", got)
		}
	})

	t.Run("tiles", func(t *testing.T) {
		atlas := m.Tiles[2]
		if atlas.Image != "img/tiles.png" || atlas.Src != image.Rect(16, 0, 32, 16) {
			t.Errorf("tile 2 = %q %v", atlas.Image, atlas.Src)
		}
		if atlas.Class != "Wall" || !atlas.Props.Bool("solid") {
			t.Errorf("tile 2 meta = %q %v", atlas.Class, atlas.Props)
		}
		if len(atlas.Colliders) != 1 || !reflect.DeepEqual(atlas.Colliders[0], tilemap.Shape{Pos: tilemap.Vector{X: 2, Y: 3}, Size: tilemap.Vector{X: 10, Y: 11}}) {
			t.Errorf("tile 2 colliders = %+v", atlas.Colliders)
		}

		anim := m.Tiles[1]
		if anim.AnimationDuration() != 400*time.Millisecond {
			t.Errorf("animation duration = %v", anim.AnimationDuration())
		}
		if anim.FrameAt(50*time.Millisecond) != 1 || anim.FrameAt(150*time.Millisecond) != 2 {
			t.Errorf("animation frames = %+v", anim.Animation)
		}

		crate := m.Tiles[5]
		if crate.Image != "img/crate.png" || crate.Src != image.Rect(0, 0, 16, 24) {
			t.Errorf("crate = %q %v", crate.Image, crate.Src)
		}
		if crate.Offset != image.Pt(1, -2) || crate.Class != "Box" {
			t.Errorf("crate meta = %v %q", crate.Offset, crate.Class)
		}

		if m.Tiles[7].Image != "img/unused.png" || m.Tiles[7].Src != image.Rect(8, 0, 16, 8) {
			t.Errorf("unused tile = %q %v", m.Tiles[7].Image, m.Tiles[7].Src)
		}
	})

	t.Run("layer order", func(t *testing.T) {
		var names []string
		for _, l := range m.Layers {
			names = append(names, l.Base().Name)
		}
		want := []string{"ground", "objs", "bg", "g"}
		if len(names) != len(want) {
			t.Fatalf("layers = %v", names)
		}
		for i := range want {
			if names[i] != want[i] {
				t.Fatalf("layers = %v, want %v", names, want)
			}
		}
	})

	t.Run("tile layers", func(t *testing.T) {
		ground := m.Layers[0].(*tilemap.TileLayer)
		want := []tilemap.Cell{{Tile: 1}, {Tile: 2}, {}, {Tile: 4}}
		for i, c := range want {
			if ground.Cells[i] != c {
				t.Errorf("ground[%d] = %+v, want %+v", i, ground.Cells[i], c)
			}
		}

		g := m.Layers[3].(*tilemap.GroupLayer)
		if g.Visible {
			t.Error("group should be invisible")
		}
		top := g.Layers[0].(*tilemap.TileLayer)
		want = []tilemap.Cell{{Tile: 1}, {Tile: 2, Flip: tilemap.FlipH}, {}, {Tile: 3}}
		for i, c := range want {
			if top.Cells[i] != c {
				t.Errorf("top[%d] = %+v, want %+v", i, top.Cells[i], c)
			}
		}
	})

	t.Run("objects", func(t *testing.T) {
		objs := m.Layers[1].(*tilemap.ObjectLayer)
		if objs.OffsetX != 5 || objs.OffsetY != 6 {
			t.Errorf("offset = %v,%v", objs.OffsetX, objs.OffsetY)
		}
		if objs.DrawOrder != tilemap.DrawOrderTopDown {
			t.Errorf("default DrawOrder = %v, want topdown", objs.DrawOrder)
		}
		if len(objs.Objects) != 5 {
			t.Fatalf("len(Objects) = %d", len(objs.Objects))
		}

		crate := objs.Objects[0]
		if crate.Tile != (tilemap.Cell{Tile: 5, Flip: tilemap.FlipH}) || crate.Type != "Box" || !crate.Visible {
			t.Errorf("crate = %+v", crate)
		}
		if crate.Opacity != 0.5 || objs.Objects[1].Opacity != 1 {
			t.Errorf("opacity = %v, %v, want 0.5, 1", crate.Opacity, objs.Objects[1].Opacity)
		}
		if crate.Pos != (tilemap.Vector{X: 32, Y: 24}) {
			t.Errorf("crate top-left = %v, want {32 24}", crate.Pos)
		}

		zone := objs.Objects[1]
		if zone.Shape.Kind != tilemap.ShapePolygon || len(zone.Shape.Points) != 3 || zone.Shape.Points[1] != (tilemap.Vector{X: 4}) {
			t.Errorf("zone = %+v", zone.Shape)
		}
		if objs.Objects[2].Shape.Kind != tilemap.ShapePoint {
			t.Errorf("spawn = %+v", objs.Objects[2].Shape)
		}
		if objs.Objects[3].Shape.Kind != tilemap.ShapeEllipse {
			t.Errorf("ellipse = %+v", objs.Objects[3].Shape)
		}

		tpl := objs.Objects[4]
		if tpl.Name != "tpl-crate" || tpl.Tile.Tile != 5 || tpl.Size != (tilemap.Vector{X: 16, Y: 24}) || tpl.Pos.X != 64 {
			t.Errorf("template object = %+v", tpl)
		}
		if tpl.Props.Int("hp") != 9 || tpl.Props.String("kind") != "wood" {
			t.Errorf("template props = %v", tpl.Props)
		}
	})

	t.Run("image layer", func(t *testing.T) {
		bg := m.Layers[2].(*tilemap.ImageLayer)
		if bg.Image != "img/bg.png" || bg.Opacity != 0.5 || bg.OffsetX != 1 || bg.OffsetY != 2 {
			t.Errorf("bg = %+v", bg)
		}
	})
}

func anchorMap(orientation, alignment, object string) fstest.MapFS {
	return fstest.MapFS{"m.tmx": {Data: []byte(`<map orientation="` + orientation + `" width="1" height="1" tilewidth="16" tileheight="16" infinite="0">
 <tileset firstgid="1" name="t" tilewidth="16" tileheight="8" tilecount="1" columns="1" objectalignment="` + alignment + `">
  <image source="t.png" width="16" height="8"/>
 </tileset>
 <layer id="1" name="l" width="1" height="1"><data encoding="csv">0</data></layer>
 <objectgroup id="2" name="o">` + object + `</objectgroup>
</map>`)}}
}

func TestTileObjectAnchor(t *testing.T) {
	cases := []struct {
		name, orientation, alignment, object string
		x, y, w, h                           float64
	}{
		{"ortho default", "orthogonal", "unspecified", `<object id="1" gid="1" x="100" y="50" width="16" height="8"/>`, 100, 42, 16, 8},
		{"iso default", "isometric", "unspecified", `<object id="1" gid="1" x="100" y="50" width="16" height="8"/>`, 92, 42, 16, 8},
		{"topleft", "orthogonal", "topleft", `<object id="1" gid="1" x="100" y="50" width="16" height="8"/>`, 100, 50, 16, 8},
		{"center scaled", "orthogonal", "center", `<object id="1" gid="1" x="100" y="50" width="32" height="16"/>`, 84, 42, 32, 16},
		{"no size", "orthogonal", "unspecified", `<object id="1" gid="1" x="100" y="50"/>`, 100, 42, 16, 8},
		// Bottom-left anchor rotated 90° clockwise: the top-left corner swings to the right of it.
		{"rotated", "orthogonal", "unspecified", `<object id="1" gid="1" x="100" y="50" width="16" height="8" rotation="90"/>`, 108, 50, 16, 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Load(anchorMap(tc.orientation, tc.alignment, tc.object), "m.tmx")
			if err != nil {
				t.Fatal(err)
			}
			o := m.Layers[1].(*tilemap.ObjectLayer).Objects[0]
			if !near(o.Pos.X, tc.x) || !near(o.Pos.Y, tc.y) || o.Size != (tilemap.Vector{X: tc.w, Y: tc.h}) {
				t.Errorf("object = %v,%v %vx%v, want %v,%v %vx%v", o.Pos.X, o.Pos.Y, o.Size.X, o.Size.Y, tc.x, tc.y, tc.w, tc.h)
			}
		})
	}
}

func TestPlainObjectUnchanged(t *testing.T) {
	m, err := Load(anchorMap("orthogonal", "unspecified", `<object id="1" x="100" y="50" width="16" height="8"/>`), "m.tmx")
	if err != nil {
		t.Fatal(err)
	}
	if o := m.Layers[1].(*tilemap.ObjectLayer).Objects[0]; o.Pos != (tilemap.Vector{X: 100, Y: 50}) {
		t.Errorf("object = %v, want {100 50}", o.Pos)
	}
}

func TestDrawOrderIndex(t *testing.T) {
	fsys := anchorMap("orthogonal", "unspecified", "")
	fsys["m.tmx"].Data = []byte(strings.Replace(string(fsys["m.tmx"].Data), `name="o"`, `name="o" draworder="index"`, 1))
	m, err := Load(fsys, "m.tmx")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Layers[1].(*tilemap.ObjectLayer).DrawOrder; got != tilemap.DrawOrderIndex {
		t.Errorf("DrawOrder = %v, want index", got)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestLoadErrors(t *testing.T) {
	fsys := fstest.MapFS{
		"inf.tmx": {Data: []byte(`<map orientation="orthogonal" width="1" height="1" tilewidth="1" tileheight="1" infinite="1"></map>`)},
	}
	if _, err := Load(fsys, "inf.tmx"); !errors.Is(err, ErrInfiniteMap) {
		t.Errorf("infinite: err = %v", err)
	}
	if _, err := Load(fsys, "map.tmj"); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("tmj: err = %v", err)
	}
	if _, err := Load(fsys, "missing.tmx"); err == nil {
		t.Error("missing: want error")
	}
}

func TestLoadExampleMap(t *testing.T) {
	fsys := os.DirFS("../examples/coins/assets")
	m, err := Load(fsys, "coins.tmx")
	if err != nil {
		t.Fatal(err)
	}

	ground, ok := m.Layer("ground").(*tilemap.TileLayer)
	if !ok {
		t.Fatal("ground layer missing")
	}
	if def := m.Tile(ground.At(0, 0).Tile); def == nil || def.Class != "Wall" {
		t.Errorf("corner tile = %+v, want a Wall", def)
	}

	objects := m.Layer("objects").(*tilemap.ObjectLayer)
	if len(objects.Objects) != 6 {
		t.Fatalf("len(Objects) = %d, want 6", len(objects.Objects))
	}
	if p := objects.Objects[0]; p.Type != "Player" || p.Pos != (tilemap.Vector{X: 32, Y: 32}) {
		t.Errorf("player = %+v, want top-left 32,32", p)
	}

	for i, def := range m.Tiles[1:] {
		if _, err := fs.Stat(fsys, string(def.Image)); err != nil {
			t.Errorf("tile %d image: %v", i+1, err)
		}
	}
}
