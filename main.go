package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func createEncyptionKey(password []byte) *[32]byte {
	key := [32]byte{}
	h := sha256.New()
	h.Write(password)
	copy(key[:], h.Sum(nil))
	return &key
}

// encrypts data using 256-bit AES-GCM
func encrypt(plaintext []byte, key *[32]byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypts data using 256-bit AES-GCM.
func decrypt(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}

// printBits is a quick way to get binary representation of a value
func printBits(n uint32) {
	fmt.Println(strconv.FormatUint(uint64(n), 2))
}

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

// encode takes an image and encodes a payload into the LSB
func encode(w io.Writer, r io.Reader, payload []byte) error {
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

// decode takes an image and prints the payload that was encoded
func decode(r io.Reader) ([]byte, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	cimg := image.NewRGBA(bounds)
	draw.Draw(cimg, bounds, img, image.Point{}, draw.Over)

	data := []uint8{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := cimg.At(x, y).RGBA()
			data = append(data, uint8(r>>8)&1)
			data = append(data, uint8(g>>8)&1)
			data = append(data, uint8(b>>8)&1)
		}
	}
	out := assemble(data)
	return out, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("encode or decode subcommand is required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "encode":
		encodeCommand := flag.NewFlagSet("encode", flag.ExitOnError)
		secret := encodeCommand.String("secret", "", "The secret to be used when encrypting your payload (required)")
		textPayload := encodeCommand.String("text", "", "text you want to encode into the image (required)")
		encInputFile := encodeCommand.String("input", "", "base image file used to store your payload (required)")
		encOutputFile := encodeCommand.String("output", "", "output destination for your new image (required)")
		encodeCommand.Parse(os.Args[2:])
		if *encInputFile == "" || *encOutputFile == "" || *textPayload == "" || *secret == "" {
			encodeCommand.PrintDefaults()
			os.Exit(2)
		}
		input, _ := filepath.Abs(*encInputFile)
		output, _ := filepath.Abs(*encOutputFile)
		fmt.Println(input, output)
		reader, err := os.Open(input)
		if err != nil {
			log.Fatal(err)
		}

		writer, err := os.Create(output)
		if err != nil {
			log.Fatal(err)
		}
		defer writer.Close()
		key := createEncyptionKey([]byte(*secret))
		payload, err := encrypt([]byte(*textPayload), key)
		if err != nil {
			log.Fatal(err)
		}
		err = encode(writer, reader, payload)
		if err != nil {
			log.Fatal(err)
		}
	case "decode":
		decodeCommand := flag.NewFlagSet("decode", flag.ExitOnError)
		secret := decodeCommand.String("secret", "", "The secret to be used when encrypting your payload (required)")
		decInputFile := decodeCommand.String("input", "", "image file where payload is thought to be")
		decodeCommand.Parse(os.Args[2:])
		input, _ := filepath.Abs(*decInputFile)
		reader, err := os.Open(input)
		if err != nil {
			log.Fatal(err)
		}
		payload, err := decode(reader)
		if err != nil {
			log.Fatal(err)
		}
		key := createEncyptionKey([]byte(*secret))
		payload, err = decrypt([]byte(payload), key)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(payload))
	default:
		fmt.Println("encode or decode subcommand is required")
		os.Exit(2)
	}
}
