package main

import (
	"image/jpeg"
	"io"
	"log"
	"os"
	"strings"

	"github.com/adrium/goheif"
	"github.com/urfave/cli/v2"
)

// Skip Writer for exif writing
type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func main() {
	app := &cli.App{
		Name:  "heic2jpeg is a simple tool to convert a HEIC-image to the JPEG-format",
		Usage: "heic2jpeg <filename.heic>\" | Example: heic2jpeg \"holiday.heic\"\n",
		Action: func(cCtx *cli.Context) error {
			err := convertHeicToJpg(cCtx.Args().First(), strings.ReplaceAll(cCtx.Args().First(), ".HEIC", ".jpg"))
			if err != nil {
				log.Fatal(err)
			}

			log.Println("Conversion Passed")

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// convertHeicToJpg takes in an input file (of heic format) and converts
// it to a jpeg format, named as the output parameters.
func convertHeicToJpg(input, output string) error {

	fileInput, err := os.Open(input)
	if err != nil {
		return err
	}
	defer fileInput.Close()

	// Extract exif to add back in after conversion
	exif, err := goheif.ExtractExif(fileInput)
	if err != nil {
		return err
	}

	img, err := goheif.Decode(fileInput)
	if err != nil {
		return err
	}

	fileOutput, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fileOutput.Close()

	// Write both convert file + exif data back
	w, _ := newWriterExif(fileOutput, exif)
	err = jpeg.Encode(w, img, nil)
	if err != nil {
		return err
	}

	return nil
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen < w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1Marker := 0xe1
		markerlen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1Marker), uint8(markerlen >> 8), uint8(markerlen & 0xff)}
		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}

	return writer, nil
}
