package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"sync/atomic"
)

const maxFileSize = 8_000_000

// TODO: Upload documents MIME types:
// ['application/pdf', 'application/rtf', 'application/msword', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document']

type uploadResult struct {
	Hash string `json:"hash"`
	Ext  string `json:"ext"`
	URI  string `json:"uri"`
	Mime string `json:"mime"`
	Sig  string `json:"signature"`
}

// ImageUploadHandler uploads images
type ImageUploadHandler struct {
	UploadTypes map[string]bool
	ImagePath   func(string, string, string) string
	uriHelper   func(string) string
	uploadKey   string
}

// UploadLocal - copy local file as if it was uploaded
func (i *ImageUploadHandler) UploadLocal(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["name"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'name' is missing")
		return
	}

	file, err := os.Open(keys[0])
	if err != nil {
		writeError(w, err.Error(), 11)
		return
	}

	defer file.Close()

	ext, mimeType, err := getFtype(file)
	if err != nil {
		writeError(w, err.Error(), 13)
		return
	}

	if _, ok := i.UploadTypes[ext]; !ok {
		writeError(w, fmt.Sprintf("Недопустимый тип файла: %v", mimeType), 14)
		return
	}

	fname, err := genName(file)
	if err != nil {
		writeError(w, err.Error(), 15)
		return
	}
	err = i.saveUploadedFile(file, i.ImagePath(OriginalSize, fname, ext))
	if err != nil {
		writeError(w, err.Error(), 16)
		return
	}
	uri := i.uriHelper(fname)

	res := uploadResult{
		fname,
		ext,
		uri,
		mimeType,
		genSignature(uri, i.uploadKey),
	}
	atomic.AddInt64(&stats.Uploaded, 1)
	writeResult(w, res)
}

// ServeHTTP - upload image file
func (i *ImageUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed:"+r.Method, 405)
		return
	}

	err := r.ParseMultipartForm(16 << 19) // 8Mb
	if err != nil {
		writeError(w, err.Error(), 406)
		return
	}
	file, header, err := r.FormFile("upload")
	if err != nil {
		writeError(w, err.Error(), 11)
		return
	}
	if header.Size > maxFileSize {
		writeError(w, fmt.Sprintf("Превышен допустимый размер файла(%v): %v", maxFileSize, header.Size), 12)
		return
	}
	defer file.Close()

	ext, mimeType, err := getFtype(file)
	if err != nil {
		writeError(w, err.Error(), 13)
		return
	}

	if _, ok := i.UploadTypes[ext]; !ok {
		writeError(w, fmt.Sprintf("Недопустимый тип файла: %v", mimeType), 14)
		return
	}

	fname, err := genName(file)
	if err != nil {
		writeError(w, err.Error(), 15)
		return
	}
	err = i.saveUploadedFile(file, i.ImagePath(OriginalSize, fname, ext))
	if err != nil {
		writeError(w, err.Error(), 16)
		return
	}
	uri := i.uriHelper(fname)
	res := uploadResult{
		fname,
		ext,
		uri,
		mimeType,
		genSignature(uri, i.uploadKey),
	}
	atomic.AddInt64(&stats.Uploaded, 1)
	writeResult(w, res)
}

func getFtype(file io.ReadSeeker) (ext, mimeType string, err error) {
	defer file.Seek(0, io.SeekStart)

	buffer := make([]byte, 512)

	_, err = file.Read(buffer)
	if err != nil {
		return
	}
	mimeType = http.DetectContentType(buffer)
	extslice, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return
	}
	if extslice != nil {
		ext = extslice[0]
	}
	return
}

func genName(file multipart.File) (string, error) {
	//Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	defer file.Seek(0, io.SeekStart)
	//Open a new hash interface to write to
	hash := md5.New()

	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	//Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil
}

func (i *ImageUploadHandler) saveUploadedFile(file multipart.File, to string) error {
	defer file.Seek(0, io.SeekStart)

	f, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		fmt.Println(err)
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, file)
	return err
}

func genSignature(data, secret string) string {
	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(secret))

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	return hex.EncodeToString(h.Sum(nil))
}

// <form action="/upload/image" method="POST" enctype="multipart/form-data"> <input type="file" name="upload"> <input type="submit" value="Upload"></form>
