package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont" // ★ ここを使用します
	"golang.org/x/image/math/fixed"
)

func main() {
	if err := _main(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func _main() error {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	face := basicfont.Face7x13

	lines := bytes.Split(stdin, []byte("\n"))
	maxWidth := 0
	totalHeight := 0
	lineHeight := face.Metrics().Height.Ceil()

	for _, line := range lines {
		lineWidth := len(line) * 7
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
		totalHeight += lineHeight
	}

	padding := 10
	imgWidth := maxWidth + 2*padding
	imgHeight := totalHeight + 2*padding
	backgroundColor := color.Black
	textColor := color.White

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: backgroundColor}, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: face,
	}

	y := padding + face.Metrics().Ascent.Ceil()
	for _, line := range lines {
		d.Dot = fixed.Point26_6{
			X: fixed.I(padding),
			Y: fixed.I(y),
		}
		d.DrawBytes(line)
		y += lineHeight
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}
