package gofakeit

import (
	"bytes"
	"errors"
	img "image"
	imgCol "image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"strconv"
)

// ImageURL will generate a random Image Based Upon Height And Width. https://picsum.photos/
func ImageURL(width int, height int) string { return imageURL(globalFaker.Rand, width, height) }

// ImageURL will generate a random Image Based Upon Height And Width. https://picsum.photos/
func (f *Faker) ImageURL(width int, height int) string { return imageURL(f.Rand, width, height) }

func imageURL(r *rand.Rand, width int, height int) string {
	return "https://picsum.photos/" + strconv.Itoa(width) + "/" + strconv.Itoa(height)
}

// Image generates a random rgba image
func Image(width int, height int) *img.RGBA { return image(globalFaker.Rand, width, height) }

// Image generates a random rgba image
func (f *Faker) Image(width int, height int) *img.RGBA { return image(f.Rand, width, height) }

func image(r *rand.Rand, width int, height int) *img.RGBA {
	upLeft := img.Point{0, 0}
	lowRight := img.Point{width, height}

	img := img.NewRGBA(img.Rectangle{upLeft, lowRight})

	// Set color for each pixel
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, imgCol.RGBA{uint8(number(r, 0, 255)), uint8(number(r, 0, 255)), uint8(number(r, 0, 255)), 0xff})
		}
	}

	return img
}

// ImageJpeg generates a random rgba jpeg image
func ImageJpeg(width int, height int) []byte { return imageJpeg(globalFaker.Rand, width, height) }

// ImageJpeg generates a random rgba jpeg image
func (f *Faker) ImageJpeg(width int, height int) []byte { return imageJpeg(f.Rand, width, height) }

func imageJpeg(r *rand.Rand, width int, height int) []byte {
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, image(r, width, height), nil)
	return buf.Bytes()
}

// ImagePng generates a random rgba png image
func ImagePng(width int, height int) []byte { return imagePng(globalFaker.Rand, width, height) }

// ImagePng generates a random rgba png image
func (f *Faker) ImagePng(width int, height int) []byte { return imagePng(f.Rand, width, height) }

func imagePng(r *rand.Rand, width int, height int) []byte {
	buf := new(bytes.Buffer)
	png.Encode(buf, image(r, width, height))
	return buf.Bytes()
}

func addImageLookup() {
	AddFuncLookup("imageurl", Info{
		Display:     "Image URL",
		Category:    "image",
		Description: "Web address pointing to an image file that can be accessed and displayed online",
		Example:     "https://picsum.photos/500/500",
		Output:      "string",
		Params: []Param{
			{Field: "width", Display: "Width", Type: "int", Default: "500", Description: "Image width in px"},
			{Field: "height", Display: "Height", Type: "int", Default: "500", Description: "Image height in px"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			width, err := info.GetInt(m, "width")
			if err != nil {
				return nil, err
			}
			if width < 10 || width >= 1000 {
				return nil, errors.New("invalid image width, must be greater than 10, less than 1000")
			}

			height, err := info.GetInt(m, "height")
			if err != nil {
				return nil, err
			}
			if height < 10 || height >= 1000 {
				return nil, errors.New("invalid image height, must be greater than 10, less than 1000")
			}

			return imageURL(r, width, height), nil
		},
	})

	AddFuncLookup("imagejpeg", Info{
		Display:     "Image JPEG",
		Category:    "image",
		Description: "Image file format known for its efficient compression and compatibility",
		Example:     "file.jpeg - bytes",
		Output:      "[]byte",
		ContentType: "image/jpeg",
		Params: []Param{
			{Field: "width", Display: "Width", Type: "int", Default: "500", Description: "Image width in px"},
			{Field: "height", Display: "Height", Type: "int", Default: "500", Description: "Image height in px"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			width, err := info.GetInt(m, "width")
			if err != nil {
				return nil, err
			}
			if width < 10 || width >= 1000 {
				return nil, errors.New("invalid image width, must be greater than 10, less than 1000")
			}

			height, err := info.GetInt(m, "height")
			if err != nil {
				return nil, err
			}
			if height < 10 || height >= 1000 {
				return nil, errors.New("invalid image height, must be greater than 10, less than 1000")
			}

			return imageJpeg(r, width, height), nil
		},
	})

	AddFuncLookup("imagepng", Info{
		Display:     "Image PNG",
		Category:    "image",
		Description: "Image file format known for its lossless compression and support for transparency",
		Example:     "file.png - bytes",
		Output:      "[]byte",
		ContentType: "image/png",
		Params: []Param{
			{Field: "width", Display: "Width", Type: "int", Default: "500", Description: "Image width in px"},
			{Field: "height", Display: "Height", Type: "int", Default: "500", Description: "Image height in px"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			width, err := info.GetInt(m, "width")
			if err != nil {
				return nil, err
			}
			if width < 10 || width >= 1000 {
				return nil, errors.New("invalid image width, must be greater than 10, less than 1000")
			}

			height, err := info.GetInt(m, "height")
			if err != nil {
				return nil, err
			}
			if height < 10 || height >= 1000 {
				return nil, errors.New("invalid image height, must be greater than 10, less than 1000")
			}

			return imagePng(r, width, height), nil
		},
	})
}
