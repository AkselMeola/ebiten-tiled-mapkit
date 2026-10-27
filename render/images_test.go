package render

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"
)

func TestImages(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"ui/button.png": {Data: buf.Bytes()}}

	images := NewImages(NewImageCache(fsys)).Add("button", "ui/button.png")
	if err := images.Preload(); err != nil {
		t.Fatal(err)
	}
	if got := images.Get("button").Bounds().Dx(); got != 3 {
		t.Errorf("button width = %d, want 3", got)
	}

	images.AddRegion("corner", "ui/button.png", image.Rect(1, 0, 3, 2))
	if got := images.Get("corner").Bounds(); got != image.Rect(1, 0, 3, 2) {
		t.Errorf("corner bounds = %v", got)
	}
	images.AddRegion("outside", "ui/button.png", image.Rect(2, 0, 4, 2))
	if err := images.Preload(); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("Preload error = %v, want one for the region outside the image", err)
	}
	delete(images.entries, "outside")

	images.Add("missing", "ui/missing.png")
	if err := images.Preload(); err == nil || !strings.Contains(err.Error(), "ui/missing.png") {
		t.Errorf("Preload error = %v, want one naming ui/missing.png", err)
	}
	mustPanic(t, func() { images.Get("missing") })
	mustPanic(t, func() { images.Get("unknown") })
}

func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	f()
}
