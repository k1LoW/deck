package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	var output string
	flag.StringVar(&output, "o", "", "output file path")
	flag.Parse()

	// Create a 1x1 pixel stub PNG image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255}) // Red pixel

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		// Write to file
		if err := os.WriteFile(output, buf.Bytes(), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Write to stdout
		if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to stdout: %v\n", err)
			os.Exit(1)
		}
	}
}
