package main

import (
	"github.com/h2non/bimg"
	"io"
	"io/ioutil"
)

// ImageFile - for image processing
type ImageFile struct {
	data []byte
	t    string // image type
}

// ImageService provides operations on images.
type ImageService interface {
	Read(fname string) (*ImageFile, error)
	Write(w io.Writer) (int, error)
}

// NewImage - creates new empty imagefile
func (i ImageFile) NewImage() *ImageFile {
	return &ImageFile{[]byte{}, ""}
}

func (i *ImageFile) Read(fname string) (*ImageFile, error) {
	b, err := ioutil.ReadFile(fname)
	i.data = b
	return i, err
}
func (i *ImageFile) Convert(to string) (*ImageFile, error) {
	if i.Type() == to {
		return i, nil
	}
	i.data, err := bimg.NewImage(i.data).Convert()
}
