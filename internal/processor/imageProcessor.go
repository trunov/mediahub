package processor

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ImageModifier defines an image modifier
type ImageModifier interface {
	Modify(img image.Image) image.Image
}

// ImageResizer defines image resizer
type ImageResizer struct {
	Width  int
	Height int
}

// Modify to implement ImageModifier interface
func (r *ImageResizer) Modify(img image.Image) image.Image {
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	if w == 0 || h == 0 || (r.Width == 0 && r.Height == 0) {
		return img
	}

	ratio := w / float64(r.Width)
	if hRatio := h / float64(r.Height); hRatio > ratio {
		ratio = hRatio
	}

	// Nothing to do - return original image
	if ratio <= 1 {
		return img
	}

	return imaging.Resize(img, int(w/ratio), int(h/ratio), imaging.Lanczos)
}

// LoadImage reads image from reader and applies requested modifiers to that image
func LoadImage(r io.Reader, modifiers ...ImageModifier) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		img = modifier.Modify(img)
	}

	return img, nil
}

// Load images, apply actions on them and then encode
type ImageProcessor struct {
	img image.Image
}

func (i *ImageProcessor) LoadPNG(r io.Reader) error {
	img, err := png.Decode(r)
	i.img = img
	return err
}

func (i *ImageProcessor) LoadJPEG(r io.Reader) error {
	img, err := jpeg.Decode(r)
	i.img = img

	return err
}
func (i *ImageProcessor) LoadWEBP(r io.Reader) error {
	img, err := webp.Decode(r)
	i.img = img

	return err

}

func (i *ImageProcessor) Resize(height int, width int) {
	i.img = imaging.Resize(i.img, width, height, imaging.Lanczos)
}

func (i *ImageProcessor) GetPNG() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, i.img)
	return buf.Bytes(), err
}

func (i *ImageProcessor) GetJPEG() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, i.img, &jpeg.Options{Quality: 90})
	return buf.Bytes(), err
}
func (i *ImageProcessor) GetWEBP() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := webp.Encode(buf, i.img, &webp.Options{
		Lossless: false,
		Quality:  90,
		Exact:    true,
	})
	return buf.Bytes(), err
}

func (i *ImageProcessor) GetBounds() (int, int) {
	return i.img.Bounds().Size().X, i.img.Bounds().Size().Y
}
