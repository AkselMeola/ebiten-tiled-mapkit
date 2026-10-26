//go:build ignore

// Generates tiles.png: go run gen.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const size = 32

var (
	grass     = color.RGBA{0x6a, 0xb0, 0x4c, 0xff}
	grassDark = color.RGBA{0x58, 0x96, 0x3e, 0xff}
	stone     = color.RGBA{0x8a, 0x8a, 0x94, 0xff}
	mortar    = color.RGBA{0x5c, 0x5c, 0x66, 0xff}
	water     = color.RGBA{0x3a, 0x7c, 0xd0, 0xff}
	foam      = color.RGBA{0xa8, 0xd8, 0xf8, 0xff}
	body      = color.RGBA{0xe8, 0x6a, 0x3a, 0xff}
	eye       = color.RGBA{0x20, 0x20, 0x28, 0xff}
	white     = color.RGBA{0xff, 0xff, 0xff, 0xff}
	gold      = color.RGBA{0xf8, 0xc8, 0x30, 0xff}
	goldDark  = color.RGBA{0xc0, 0x90, 0x10, 0xff}
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 4*size, 2*size))
	tiles := []func(x, y int) color.Color{
		grassTile,
		wallTile,
		waterTile(0),
		waterTile(6),
		playerTile(0),
		playerTile(2),
		coinTile(10),
		coinTile(4),
	}
	for i, tile := range tiles {
		ox, oy := (i%4)*size, (i/4)*size
		for y := range size {
			for x := range size {
				if c := tile(x, y); c != nil {
					img.Set(ox+x, oy+y, c)
				}
			}
		}
	}

	f, err := os.Create("tiles.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func grassTile(x, y int) color.Color {
	if (x*7+y*13)%23 == 0 {
		return grassDark
	}
	return grass
}

func wallTile(x, y int) color.Color {
	row := y / 8
	shift := 0
	if row%2 == 1 {
		shift = 8
	}
	if y%8 == 7 || (x+shift)%16 == 15 {
		return mortar
	}
	return stone
}

func waterTile(phase int) func(x, y int) color.Color {
	return func(x, y int) color.Color {
		wave := int(2 * math.Sin(float64(x+phase)*math.Pi/8))
		if (y+wave)%10 == 0 {
			return foam
		}
		return water
	}
}

func playerTile(lift int) func(x, y int) color.Color {
	return func(x, y int) color.Color {
		y += lift
		dx, dy := float64(x)-15.5, float64(y)-17.5
		switch {
		case inCircle(x, y, 20, 13, 3):
			return white
		case inCircle(x, y, 21, 13, 1.5):
			return eye
		case dx*dx/121+dy*dy/144 <= 1:
			return body
		}
		return nil
	}
}

func coinTile(halfWidth float64) func(x, y int) color.Color {
	return func(x, y int) color.Color {
		dx, dy := float64(x)-15.5, float64(y)-15.5
		d := dx*dx/(halfWidth*halfWidth) + dy*dy/100
		switch {
		case d <= 0.55:
			return gold
		case d <= 1:
			return goldDark
		}
		return nil
	}
}

func inCircle(x, y int, cx, cy, r float64) bool {
	dx, dy := float64(x)-cx, float64(y)-cy
	return dx*dx+dy*dy <= r*r
}
