package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
)

const maxFileSize = 8_000_000

type uploadResult struct {
	Hash string `json:"hash"`
	Ext  string `json:"ext"`
	URI  string `json:"uri"`
	Mime string `json:"mime"`
}

// Upload - upload image file
func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", 405)
		return
	}
	r.ParseMultipartForm(16 << 19) // 8Mb
	file, header, err := r.FormFile("upload")
	if err != nil {
		writeError(w, err.Error(), 400)
		return
	}
	if header.Size > maxFileSize {
		writeError(w, fmt.Sprintf("Превышен допустимый размер файла(%v): %v", maxFileSize, header.Size), 400)
		return
	}
	defer file.Close()

	ext, err := getFtype(file)
	if err != nil {
		writeError(w, err.Error(), 500)
		return
	}

	if !allowedTypes[ext] {
		writeError(w, fmt.Sprintf("Недопустимый тип файла: %v", ext), 500)
		return
	}

	fname, err := genName(file)
	if err != nil {
		writeError(w, err.Error(), 500)
		return
	}
	err = saveUploadedFile(file, imagePath("o", fname+ext))
	if err != nil {
		writeError(w, err.Error(), 500)
		return
	}
	res := getUploadResult(fname, ext)
	writeResult(w, res)
}

func getUploadResult(fname, ext string) uploadResult {
	return uploadResult{
		fname,
		ext,
		uriHelper(fname),
		"",
	}
}

func getFtype(file multipart.File) (string, error) {
	var ext string
	defer file.Seek(0, io.SeekStart)

	buffer := make([]byte, 512)

	_, err := file.Read(buffer)
	if err != nil {
		return ext, err
	}
	contentType := http.DetectContentType(buffer)
	extslice, err := mime.ExtensionsByType(contentType)
	if err != nil {
		return ext, err
	}
	if extslice == nil {
		return ext, nil
	}
	return extslice[0], nil
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

func saveUploadedFile(file multipart.File, to string) error {
	defer file.Seek(0, io.SeekStart)

	f, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = io.Copy(f, file)
	return err
}

// <form action="/upload/image" method="POST" enctype="multipart/form-data"> <input type="file" name="upload"> <input type="submit" value="Upload"></form>