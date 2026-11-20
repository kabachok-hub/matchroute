// Package icon renders the MatchRoute avatar and its tray status indicator.
package icon

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"math"

	"golang.org/x/image/draw"
)

var (
	Green  = color.NRGBA{0x22, 0xc5, 0x5e, 0xff}
	Yellow = color.NRGBA{0xea, 0xb3, 0x08, 0xff}
	Gray   = color.NRGBA{0x94, 0xa3, 0xb8, 0xff}
	None   = color.NRGBA{}
)

// avatar.png is the application-sized copy of docs/img/matchroute-avatar.png.
//
//go:embed avatar.png
var avatarPNG []byte

var avatar = func() image.Image {
	img, err := png.Decode(bytes.NewReader(avatarPNG))
	if err != nil {
		panic(err)
	}
	return img
}()

// Draw scales the avatar and optionally overlays the status dot used in the tray.
func Draw(size int, dot color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(img, img.Bounds(), avatar, avatar.Bounds(), draw.Src, nil)
	if dot == None {
		return img
	}
	center := float64(size) * 0.80
	radius := float64(size) * 0.135
	border := math.Max(float64(size)*0.04, 1)
	outline := color.NRGBA{0x08, 0x12, 0x26, 0xff}
	for y := max(0, int(center-radius-border)); y < size; y++ {
		for x := max(0, int(center-radius-border)); x < size; x++ {
			d := math.Hypot(float64(x)+0.5-center, float64(y)+0.5-center)
			switch {
			case d <= radius:
				img.SetNRGBA(x, y, dot)
			case d <= radius+border:
				img.SetNRGBA(x, y, outline)
			}
		}
	}
	return img
}

// PNG returns the icon as a PNG file.
func PNG(size int, dot color.NRGBA) []byte {
	var b bytes.Buffer
	png.Encode(&b, Draw(size, dot))
	return b.Bytes()
}
