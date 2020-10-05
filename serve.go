package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	m "mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"

	"github.com/disintegration/imaging"
)

// AllowedSizes - a map holding arrays [w, h] keyed with predefined string values
type AllowedSizes map[string][2]uint16

// ReadTypes map of bool keyed by allowed file extensions eg: .jpg
type ReadTypes map[string]bool

// ImageServe handler
type ImageServe struct {
	Sizes     AllowedSizes
	Types     ReadTypes
	imagePath func(string, string, string) string
	Domain    string
}

// ServeImage - handlerfunc for serving images
func (i *ImageServe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var mime string
	name, ext, sz, err := i.processImagePath(r.URL.EscapedPath())
	if err != nil {
		writeError(w, err.Error(), 1)
		return
	}

	fpath := i.imagePath(sz, name, ext)
	f, err := os.Open(fpath)

	// original size file
	if sz == OriginalSize {
		if err != nil {
			writeError(w, err.Error(), 2)
			return
		}
		defer f.Close()
		_, mime, err = getFtype(f)
		if err != nil {
			writeError(w, err.Error(), 3)
			return
		}
	}

	// other sizes
	if err != nil {
		if !os.IsNotExist(err) {
			writeError(w, err.Error(), 5)
			return
		}

		original, err := i.getOriginalImage(name)
		if err != nil {
			writeError(w, err.Error(), 66)
			return
		}
		wh := i.Sizes[sz]
		/*
			Imaging supports image resizing using various resampling filters. The most notable ones:

			Lanczos - A high-quality resampling filter for photographic images yielding sharp results.
			CatmullRom - A sharp cubic filter that is faster than Lanczos filter while providing similar results.
			MitchellNetravali - A cubic filter that produces smoother results with less ringing artifacts than CatmullRom.
			Linear - Bilinear resampling filter, produces smooth output. Faster than cubic filters.
			Box - Simple and fast averaging filter appropriate for downscaling. When upscaling it's similar to NearestNeighbor.
			NearestNeighbor - Fastest resampling filter, no antialiasing.
		*/
		src := imaging.Resize(original, int(wh[0]), 0, imaging.Lanczos)

		original = nil

		err = imaging.Save(src, fpath)
		if err != nil {
			writeError(w, err.Error(), 9)
			return
		}

		f, err = os.Open(fpath)
		if err != nil {
			writeError(w, err.Error(), 10)
			return
		}
		defer f.Close()

	}
	mime = m.TypeByExtension(ext)

	w.Header().Set("Content-Type", mime)
	io.Copy(w, f)
	atomic.AddInt64(&stats.Served, 1)
}
func (i *ImageServe) processImagePath(p string) (name, ext, sz string, err error) {
	fname := path.Base(p)
	parts := strings.Split(fname, ".")
	if len(parts) != 3 {
		err = errors.New("bad name")
		return
	}
	ext = path.Ext(fname)
	if ext == ".jpeg" || ext == ".jpe" {
		ext = ".jpg"
	}
	if _, ok := i.Types[ext]; !ok {
		err = errors.New("bad image type requested")
		return
	}
	if len(parts[1]) < 5 {
		err = errors.New("bad image name requested")
		return
	}
	sz = parts[0]
	if _, ok := i.Sizes[sz]; !ok {
		err = errors.New("bad image size requested")
		return
	}
	name = parts[1]
	return
}

func (i *ImageServe) getOriginalImage(fname string) (image.Image, error) {
	ext := path.Ext(fname)
	name := strings.TrimSuffix(fname, ext)
	src, err := imaging.Open(i.imagePath("o", name, ext))

	if err == nil {
		return src, nil
	}

	return nil, errors.New("Image Not found")
}

// URIHelper returns URI template for hash name
func (i *ImageServe) URIHelper(name string) string {
	return fmt.Sprintf("//%s/images/[size].%s.[ext]", i.Domain, name)
}

// URIFromHash handler for /urifrom/:hash
func (i *ImageServe) URIFromHash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	p := r.URL.EscapedPath()
	_, hash := path.Split(p)
	if stats, err := os.Stat(i.imagePath(OriginalSize, hash, "")); err != nil || stats.IsDir() {
		http.Error(w, err.Error(), 404)
		return
	}
	io.WriteString(w, i.URIHelper(hash))
}
