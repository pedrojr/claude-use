//go:build ignore

// mkicon builds assets/claude-use.ico from the tray icon (assets/tray.png).
// The tray icon is pixel art, so it is scaled with nearest neighbour to keep it sharp.
// Run from the repository root after changing tray.png:
//
//	go run assets/mkicon.go
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"log"
	"os"
)

var sizes = []int{16, 24, 32, 48, 64, 128, 256}

func main() {
	f, err := os.Open("assets/tray.png")
	if err != nil {
		log.Fatal(err)
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Each entry is stored as PNG (supported by Windows since Vista).
	var images [][]byte
	for _, s := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, scale(src, s)); err != nil {
			log.Fatal(err)
		}
		images = append(images, buf.Bytes())
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))}) // ICONDIR
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := uint8(s) // 256 is written as 0
		binary.Write(&out, binary.LittleEndian, struct {
			W, H, Colors, Reserved uint8
			Planes, BitCount       uint16
			Size, Offset           uint32
		}{dim, dim, 0, 0, 1, 32, uint32(len(images[i])), uint32(offset)})
		offset += len(images[i])
	}
	for _, b := range images {
		out.Write(b)
	}
	if err := os.WriteFile("assets/claude-use.ico", out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}

func scale(src image.Image, size int) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.Set(x, y, src.At(b.Min.X+x*b.Dx()/size, b.Min.Y+y*b.Dy()/size))
		}
	}
	return dst
}
