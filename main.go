package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jroyal/stegano/stegano"
)

// getReader is a helper function to take a filepath and return a reader
func getReader(path string) *os.File {
	p, _ := filepath.Abs(path)
	reader, err := os.Open(p)
	if err != nil {
		log.Fatal(err)
	}
	return reader
}

// getWriter is a helper function to take a filepath and return a writer
func getWriter(path string) *os.File {
	p, _ := filepath.Abs(path)
	writer, err := os.Create(p)
	if err != nil {
		log.Fatal(err)
	}
	return writer
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
		input := encodeCommand.String("input", "", "base image file used to store your payload (required)")
		output := encodeCommand.String("output", "", "output destination for your new image (required)")
		encodeCommand.Parse(os.Args[2:])
		if *input == "" || *output == "" || *textPayload == "" || *secret == "" {
			encodeCommand.PrintDefaults()
			os.Exit(2)
		}
		reader := getReader(*input)
		writer := getWriter(*output)
		defer writer.Close()
		err := stegano.Encode(writer, reader, []byte(*secret), []byte(*textPayload))
		if err != nil {
			log.Fatal(err)
		}
	case "decode":
		decodeCommand := flag.NewFlagSet("decode", flag.ExitOnError)
		secret := decodeCommand.String("secret", "", "The secret to be used when encrypting your payload (required)")
		input := decodeCommand.String("input", "", "image file where payload is thought to be (required)")
		decodeCommand.Parse(os.Args[2:])
		if *input == "" || *secret == "" {
			decodeCommand.PrintDefaults()
			os.Exit(2)
		}
		reader := getReader(*input)
		payload, err := stegano.Decode(reader, []byte(*secret))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(payload))
	default:
		fmt.Println("encode or decode subcommand is required")
		os.Exit(2)
	}
}
