package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

var iconSizes = []int{16, 24, 32, 48, 64, 128, 256}

type iconDirEntry struct {
	Width       byte
	Height      byte
	ColorCount  byte
	Reserved    byte
	Planes      uint16
	BitCount    uint16
	BytesInRes  uint32
	ImageOffset uint32
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: iconbuilder input.png output.ico")
		os.Exit(2)
	}
	input, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer input.Close()

	source, _, err := image.Decode(input)
	if err != nil {
		panic(err)
	}

	images := make([][]byte, 0, len(iconSizes))
	for _, size := range iconSizes {
		target := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, target); err != nil {
			panic(err)
		}
		images = append(images, encoded.Bytes())
	}

	var output bytes.Buffer
	_ = binary.Write(&output, binary.LittleEndian, uint16(0))
	_ = binary.Write(&output, binary.LittleEndian, uint16(1))
	_ = binary.Write(&output, binary.LittleEndian, uint16(len(images)))

	offset := uint32(6 + 16*len(images))
	for index, data := range images {
		sizeByte := byte(iconSizes[index])
		if iconSizes[index] == 256 {
			sizeByte = 0
		}
		entry := iconDirEntry{
			Width: sizeByte, Height: sizeByte,
			Planes: 1, BitCount: 32,
			BytesInRes: uint32(len(data)), ImageOffset: offset,
		}
		if err := binary.Write(&output, binary.LittleEndian, entry); err != nil {
			panic(err)
		}
		offset += uint32(len(data))
	}
	for _, data := range images {
		_, _ = output.Write(data)
	}
	if err := os.WriteFile(os.Args[2], output.Bytes(), 0o644); err != nil {
		panic(err)
	}
}
