package imaging

import (
	"image"
	"math"

	"github.com/disintegration/imaging"
	resizer "github.com/disintegration/imaging"
)

// ResizeTarget resizes to fit in the given bounds
func ResizeTarget(img image.Image, width, height int) (out image.Image) {
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	scale := 1.0
	rw, rh := math.Abs(float64(width)/w), math.Abs(float64(height)/h)
	scale = float64(height) / h
	if rw < rh {
		scale = float64(width) / w
	}
	return resizer.Resize(img, int(w*scale), int(h*scale), imaging.Lanczos)
}
