package webp_converter

import (
	"bytes"
	"fmt"
	"image"
	"io"

	"github.com/chai2010/webp"
)

type Converter struct{}

func (Converter) ToWebP(reader io.Reader, ext string) ([]byte, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	var buf bytes.Buffer
	var q float32

	if ext == ".png" {
		q = 100
	} else {
		q = 75
	}

	if err := webp.Encode(&buf, img, &webp.Options{Quality: q}); err != nil {
		return nil, fmt.Errorf("error encoding to webp: %v", err)
	}

	return buf.Bytes(), nil
}
