package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	m "mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	// "github.com/h2non/bimg" // lib vips is required! install it first: sudo apt install libvips libvips-dev
	"github.com/disintegration/imaging"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

// Config - main app configuration
// var imageDir = "images"
// var domain = "media.metring.com"
type Config struct {
	Domain       string               `yaml:"domain" envconfig:"IMAGESERVER_DOMAIN"`
	ImageDir     string               `yaml:"imgdir" envconfig:"IMAGESERVER_DIR"`
	ListenAddr   string               `yaml:"listen"  envconfig:"IMAGESERVER_LISTEN_ADDR"`
	Secret       string               `yaml:"secret"  envconfig:"APP_SECRET"`
	Debug        bool                 `yaml:"debug" envconfig:"IMAGESERVER_DEBUG"`
	UploadTypes  map[string]bool      `yaml:"uploadable" envconfig:"IMAGESERVER_UPLOAD_TYPES"`
	ReadTypes    map[string]bool      `yaml:"readable" envconfig:"IMAGESERVER_READ_TYPES"`
	AllowedSizes map[string][2]uint16 `yaml:"sizes"`
}

// Stats - global app stats
type Stats struct {
	Since    string        `json:"since"`
	Served   int64         `json:"served"`
	Uploaded int64         `json:"uploaded"`
	Errors   map[int]int64 `json:"errors"`
}

var statsMu sync.Mutex

var config Config
var stats Stats

func init() {
	stats.Since = time.Now().String()
	stats.Errors = map[int]int64{}
	config = Config{
		Domain:      "",
		ImageDir:    "images",
		ListenAddr:  "0.0.0.0:5000",
		Secret:      "",
		Debug:       false,
		UploadTypes: map[string]bool{".jpg": true, ".jpeg": true, ".png": true},
		ReadTypes:   map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true},
		AllowedSizes: map[string][2]uint16{
			"o":  [2]uint16{0, 0},
			"d":  [2]uint16{16, 9},
			"xs": [2]uint16{64, 36},
			"sm": [2]uint16{384, 216},
			"md": [2]uint16{768, 432},
			"lg": [2]uint16{1280, 720},
			"xl": [2]uint16{1920, 1080},
		},
	}
	// res, _ := yaml.Marshal(&config)
	// fmt.Println(string(res))
}

func main() {
	readConfigFile(&config)
	readConfigEnv(&config)
	if config.Domain == "" {
		log.Fatalln("config.Domain unspecified, can't continue.")
	}
	if config.Secret == "" {
		log.Fatalln("config.Secret unspecified, can't continue.")
	}

	// initialise image directories
	for k := range config.AllowedSizes {
		dir := fmt.Sprintf("%s/%s", config.ImageDir, k)
		_, err := os.Open(dir)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(dir, os.ModePerm)
				if err != nil {
					log.Fatalln(err)
				}
				log.Println("Directory created: ", dir)
			} else {
				log.Fatalln(err)
			}
		}
	}
	fmt.Println("Domain is set to", config.Domain)
	fmt.Println("Listening", config.ListenAddr)
	fmt.Println("Serving from:", config.ImageDir)
	fmt.Println("Upload allowed for", config.UploadTypes)

	http.HandleFunc("/", NotFound)
	http.HandleFunc("/urifromhash/", uriFromHash)
	http.HandleFunc("/upload/image", Upload)
	http.HandleFunc("/stats", Report)
	http.HandleFunc("/images/", ServeImage)
	log.Fatalln(http.ListenAndServe(config.ListenAddr, nil))
}

// NotFound - show error message
func NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, "Not found", 404)
}

// Report - show short stats about running server
func Report(w http.ResponseWriter, r *http.Request) {
	writeResult(w, stats)
}

// ServeImage - handlerfunc for serving images
func ServeImage(w http.ResponseWriter, r *http.Request) {
	var mime string
	name, ext, sz, err := processImagePath(r.URL.EscapedPath())
	if err != nil {
		writeError(w, err.Error(), 1)
		return
	}

	fpath := imagePath(sz, name, ext)
	f, err := os.Open(fpath)

	// original size file
	if sz == "o" {
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

		original, err := getOriginalImage(name)
		if err != nil {
			writeError(w, err.Error(), 6)
			return
		}

		wh := config.AllowedSizes[sz]

		src := imaging.Resize(original, int(wh[0]), 0, imaging.Lanczos)
		/*
			Imaging supports image resizing using various resampling filters. The most notable ones:

			Lanczos - A high-quality resampling filter for photographic images yielding sharp results.
			CatmullRom - A sharp cubic filter that is faster than Lanczos filter while providing similar results.
			MitchellNetravali - A cubic filter that produces smoother results with less ringing artifacts than CatmullRom.
			Linear - Bilinear resampling filter, produces smooth output. Faster than cubic filters.
			Box - Simple and fast averaging filter appropriate for downscaling. When upscaling it's similar to NearestNeighbor.
			NearestNeighbor - Fastest resampling filter, no antialiasing.
		*/

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

// func getOriginalImage(fname string) (*bimg.Image, error) {
// 	ext := path.Ext(fname)
// 	name := strings.TrimSuffix(fname, ext)
// 	buffer, err := bimg.Read(imagePath("o", name, ext))

// 	if err == nil {
// 		return bimg.NewImage(buffer), nil
// 	}

// 	return nil, errors.New("Image Not found")
// }
func getOriginalImage(fname string) (image.Image, error) {
	ext := path.Ext(fname)
	name := strings.TrimSuffix(fname, ext)
	src, err := imaging.Open(imagePath("o", name, ext))

	if err == nil {
		return src, nil
	}

	return nil, errors.New("Image Not found")
}
func imagePath(sz, name, ext string) string {
	if sz == "o" {
		return fmt.Sprintf("%s/%s/%s", config.ImageDir, sz, name) // сохраняем оригинальный размер без расширения
	}
	return fmt.Sprintf("%s/%s/%s%s", config.ImageDir, sz, name, ext)
}

func processImagePath(p string) (name, ext, sz string, err error) {
	fname := path.Base(p)
	parts := strings.Split(fname, ".")
	if len(parts) != 3 {
		err = errors.New("bad name")
		return
	}
	ext = path.Ext(fname)
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	if _, ok := config.ReadTypes[ext]; !ok {
		err = errors.New("bad image type requested")
		return
	}
	if len(parts[1]) < 5 {
		err = errors.New("bad image name requested")
		return
	}
	sz = parts[0]
	if _, ok := config.AllowedSizes[sz]; !ok {
		err = errors.New("bad image size requested")
		return
	}
	name = parts[1]
	return
}

func uriHelper(name string) string {
	return fmt.Sprintf("//%s/images/[size].%s.[ext]", config.Domain, name)
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

func readConfigFile(cfg *Config) {
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		log.Fatalln(err)
	}
}
func readConfigEnv(cfg *Config) {
	err := envconfig.Process("", cfg)
	if err != nil {
		log.Fatalln(err)
	}
}
