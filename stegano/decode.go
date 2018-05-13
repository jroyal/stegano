package stegano

import (
	"encoding/binary"
	"image/png"
	"io"
)

// assemble takes the LSB data from a payload and reconstructes the original message
func assemble(data []uint8) []byte {
	result := []byte{}
	length := len(data)
	for i := 0; i < len(data)/8; i++ {
		b := uint8(0)
		for j := 0; j < 8; j++ {
			if i*8+j < length {
				b = b<<1 + data[i*8+j]
			}
		}
		result = append(result, byte(b))
	}
	payloadSize := binary.LittleEndian.Uint32(result[0:4])
	return result[4 : payloadSize+4]
}

// Decode takes an image and prints the payload that was encoded
func Decode(r io.Reader, secret []byte) ([]byte, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()

	data := []uint8{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			data = append(data, uint8(r>>8)&1)
			data = append(data, uint8(g>>8)&1)
			data = append(data, uint8(b>>8)&1)
		}
	}
	payload := assemble(data)
	key := createEncyptionKey(secret)
	payload, err = decrypt(payload, key)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
