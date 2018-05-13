package stegano

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
)

// setBit will set the LSB of n to the requested value
func setBit(n uint32, is1 bool) uint8 {
	n = n >> 8
	n = n & 0xFE
	if is1 {
		n = n | 0x1
	}
	return uint8(n)
}

// convertByteToBits is a helper function that takes one byte and
// returns a slice of booleans representing the binary value of that byte
func convertByteToBits(b byte) []bool {
	result := make([]bool, 8)
	for j := 0; j < 8; j++ {
		mask := byte(1 << uint(j))
		result[7-j] = b&mask>>uint(j) == 1
	}
	return result
}

// getBits returns a slice of booleans representing the binary value of data
func getBits(data []byte) []bool {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(len(data)))
	data = append(bs, data...)
	results := []bool{}
	for _, b := range data {
		results = append(results, convertByteToBits(b)...)
	}
	return results
}

// Encode takes an image and encodes a payload into the LSB
func Encode(w io.Writer, r io.Reader, secret, payload []byte) error {
	key := createEncyptionKey(secret)
	payload, err := encrypt(payload, key)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	cimg := image.NewRGBA(bounds)
	draw.Draw(cimg, bounds, img, image.Point{}, draw.Over)

	data := getBits(payload)
	dataIdx := 0
	dataLen := len(data)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := cimg.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			a8 := uint8(a >> 8)

			if dataIdx < dataLen {
				r8 = setBit(r, data[dataIdx])
				dataIdx++
			}
			if dataIdx < dataLen {
				g8 = setBit(g, data[dataIdx])
				dataIdx++
			}
			if dataIdx < dataLen {
				b8 = setBit(b, data[dataIdx])
				dataIdx++
			}
			cimg.Set(x, y, color.RGBA{r8, g8, b8, a8})
		}
	}
	return png.Encode(w, cimg)
}
