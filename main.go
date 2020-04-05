package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/h2non/bimg" // lib vips is required! install it first: sudo apt install libvips libvips-dev
)

var imageDir = "images"
var domain = "media.metring.com"
var port = "8080"
var allowedTypes = map[string]bool{
	".png":  true,
	".gif":  true,
	".webp": true,
	".jpeg": true,
	".jpg":  true,
}
var allowedSizes = map[string][]int{
	"o":  []int{0, 0},
	"d":  []int{16, 9},
	"xs": []int{64, 36},
	"sm": []int{384, 216},
	"md": []int{768, 432},
	"lg": []int{1280, 720},
	"xl": []int{1920, 1080},
}

func init() {
	if dir := os.Getenv("IMG_DIR"); dir != "" {
		imageDir = dir
	}
	fmt.Println("ImageDir:", imageDir)
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	fmt.Println("Port:", port)
	fmt.Println("User:", os.Getenv("USER"))
	fmt.Println("Working dir:", os.Getenv("PWD"))
	upl := os.Getenv("UPLOAD_TYPES")
	fmt.Println("Upload types:", strings.Fields(upl))
	fmt.Println()

	for k := range allowedSizes {
		dir := fmt.Sprintf("%s/%s", imageDir, k)
		_, err := os.Open(dir)
		if err != nil {
			if os.IsNotExist(err) {
				os.MkdirAll(dir, os.ModePerm)
				log.Println("Directory created: ", dir)
			} else {
				log.Fatalln(err)
			}
		}
	}
}
func main() {
	http.HandleFunc("/urifromhash/", uriFromHash)
	http.HandleFunc("/upload/image", upload)
	http.HandleFunc("/images/", serveImage)
	log.Fatalln(http.ListenAndServe(":"+port, nil))
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	name, ext, sz, err := processImagePath(r.URL.EscapedPath())
	if err != nil {
		writeError(w, err.Error(), 404)
		return
	}

	fpath := imagePath(sz, name, ext)
	f, err := os.Open(fpath)
	if err != nil {
		fmt.Println(err)
		if sz == "o" {
			writeError(w, err.Error(), 404)
			return
		}
		original, err := getOriginalImage(name)
		if err != nil {
			writeError(w, err.Error(), 404)
			return
		}
		wh := allowedSizes[sz]

		src, err := original.Resize(wh[0], 0)
		if err != nil {
			writeError(w, err.Error(), 404)
			return
		}
		src, err = bimg.NewImage(src).Convert(bimgType(ext))
		if err != nil {
			writeError(w, err.Error(), 404)
			return
		}
		err = bimg.Write(fpath, src)

		if err != nil {
			writeError(w, err.Error(), 404)
			return
		}
		f, err = os.Open(fpath)
	}
	defer f.Close()
	mime := mime.TypeByExtension(ext)
	if mime == "" {
		mime = "application/octet-stream"
	}

	w.Header().Set("Content-Type", mime)
	io.Copy(w, f)
}

func bimgType(ext string) bimg.ImageType {
	switch ext {
	case ".jpg":
		fallthrough
	case ".jpeg":
		return bimg.JPEG
	case ".png":
		return bimg.PNG
	case ".webp":
		return bimg.WEBP
	case ".gif":
		return bimg.GIF
	}
	return bimg.UNKNOWN
}

func getOriginalImage(fname string) (*bimg.Image, error) {
	ext := path.Ext(fname)
	name := strings.TrimSuffix(fname, ext)
	buffer, err := bimg.Read(imagePath("o", name, ext))

	if err == nil {
		return bimg.NewImage(buffer), nil
	}

	return nil, errors.New("Image Not found")
}

func imagePath(sz, name, ext string) string {
	if sz == "o" {
		return fmt.Sprintf("%s/%s/%s", imageDir, sz, name) // сохраняем оригинальный размер без расширения
	}
	return fmt.Sprintf("%s/%s/%s%s", imageDir, sz, name, ext)
}

func processImagePath(p string) (name, ext, sz string, err error) {
	fname := path.Base(p)
	parts := strings.Split(fname, ".")
	if len(parts) != 3 {
		err = errors.New("bad name")
		return
	}
	ext = path.Ext(fname)
	if !allowedTypes[ext] {
		err = errors.New("bad image type requested")
		return
	}
	if len(parts[1]) < 5 {
		err = errors.New("bad image name requested")
		return
	}
	if sz = parts[0]; allowedSizes[sz] == nil {
		err = errors.New("bad image size requested")
		return
	}
	name = parts[1]
	return
}
func uriHelper(name string) string {
	return fmt.Sprintf("//%s/images/[size].%s.[ext]", domain, name)
}

// /urifromhash/:hash
func uriFromHash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	p := r.URL.EscapedPath()
	_, hash := path.Split(p)
	if _, err := getOriginalImage(hash); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	io.WriteString(w, uriHelper(hash))
}
