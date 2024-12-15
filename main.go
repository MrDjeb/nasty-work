package main

import (
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrium/goheif"
)

// Skip Writer for exif writing
type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
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

func main() {
	// if _, err := os.Stat("result"); os.IsNotExist(err) {
	// 	err := os.Mkdir("result", 0755)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	err := filepath.Walk("files",
		func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if ".heic" == strings.ToLower(filepath.Ext(path)) {
				extension := filepath.Ext(path)

				err := convertHeicToJpg(path, path[:len(path)-len(extension)]+".jpg")
				if err != nil {
					log.Fatal(err)
				}
				os.Remove(path)
				fmt.Println(path, " | ", path[:len(path)-len(extension)]+".jpg")

			}
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}
}