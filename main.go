package main

import (
	"fmt"
	. "imagesamenu/middleware"

	"log"
	"net/http"

	// _ "net/http/pprof"
	"os"
	"sync"
	"time"

	// "github.com/h2non/bimg" // lib vips is required! install it first: sudo apt install libvips libvips-dev

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
	UploadKey    string               `yaml:"uploadkey" envconfig:"IMAGESERVER_UPLOAD_KEY"`
}

// Stats - global app stats
type Stats struct {
	Since    string        `json:"since"`
	Served   int64         `json:"served"`
	Uploaded int64         `json:"uploaded"`
	Errors   map[int]int64 `json:"errors"`
}

var statsMu sync.Mutex

var stats Stats

func init() {
	stats.Since = time.Now().String()
	stats.Errors = map[int]int64{}
	// config = Config{
	// 	Domain:      "",
	// 	ImageDir:    "images",
	// 	ListenAddr:  "localhost:8080",
	// 	Secret:      "",
	// 	Debug:       false,
	// 	UploadTypes: map[string]bool{".jpg": true, ".jpeg": true, ".png": true},
	// 	ReadTypes:   map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true},
	// 	AllowedSizes: map[string][2]uint16{
	// 		"o":  [2]uint16{0, 0},
	// 		"d":  [2]uint16{16, 9},
	// 		"xs": [2]uint16{64, 36},
	// 		"sm": [2]uint16{384, 216},
	// 		"md": [2]uint16{768, 432},
	// 		"lg": [2]uint16{1280, 720},
	// 		"xl": [2]uint16{1920, 1080},
	// 	},
	// }
	// res, _ := yaml.Marshal(&config)
	// fmt.Println(string(res))
}

func main() {
	var config Config

	readConfigFile(&config)
	readConfigEnv(&config)
	if config.Domain == "" {
		log.Fatalln("config.Domain unspecified, can't continue.")
	}
	if config.Secret == "" {
		log.Fatalln("config.Secret unspecified, can't continue.")
	}
	if config.UploadKey == "" {
		log.Fatalln("config.UploadKey unspecified, can't continue.")
	}
	var (
		imagePath = ImagePath{config.ImageDir}
		serve     = ImageServe{config.AllowedSizes, config.ReadTypes, imagePath.ImagePath, config.Domain}
		upload    = ImageUploadHandler{config.UploadTypes, imagePath.ImagePath, serve.URIHelper, config.UploadKey}
	)
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
	// .Add(NewAuthCheck(writeError))
	uploadHandler := NewExtHandler(&upload).Add(&AuthCheck{ErrorHandler: writeError}).Add(NewTokenParser(config.Secret, true, writeError))

	if config.Debug {
		uploadHandler.Add(new(ImageForm))
	}

	http.HandleFunc("/", NotFound)
	http.HandleFunc("/urifromhash/", serve.URIFromHash)
	http.HandleFunc("/local/image", upload.UploadLocal)
	http.Handle("/upload/image", uploadHandler)
	http.HandleFunc("/stats", Report)
	http.Handle("/images/", &serve)
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

// func imagePath(sz, name, ext string) string {
// 	if sz == "o" {
// 		return fmt.Sprintf("%s/%s/%s", config.ImageDir, sz, name) // сохраняем оригинальный размер без расширения
// 	}
// 	return fmt.Sprintf("%s/%s/%s%s", config.ImageDir, sz, name, ext)
// }

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
