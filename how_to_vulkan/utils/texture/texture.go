// Package texture loads a PNG into tightly-packed RGBA8 bytes for upload.
package texture

import (
	"image"
	"image/draw"
	_ "image/png"
	"os"
)

// Image is decoded RGBA8, row-major, 4 bytes/pixel.
type Image struct {
	Width, Height int
	Pixels        []byte
}

// Load decodes a PNG (or any registered format) to RGBA8.
func Load(path string) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	return &Image{Width: b.Dx(), Height: b.Dy(), Pixels: rgba.Pix}, nil
}

// Solid builds a WxH single-color RGBA image (fallback when no texture file).
func Solid(w, h int, r, g, b, a byte) *Image {
	pix := make([]byte, w*h*4)
	for i := 0; i < w*h; i++ {
		pix[i*4+0] = r
		pix[i*4+1] = g
		pix[i*4+2] = b
		pix[i*4+3] = a
	}
	return &Image{Width: w, Height: h, Pixels: pix}
}
