package middleware

import (
	"io"
	"net/http"
)

// ImageForm shows image upload form
type ImageForm struct {
	next http.Handler
}

func (i *ImageForm) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-type", "text/html")
		io.WriteString(w, `<form action="/upload/image" method="POST" enctype="multipart/form-data"> <input type="file" name="upload"> <input type="submit" value="Upload"></form>`)
		return
	}
	i.next.ServeHTTP(w, r)
}

// Next - wrap http.handler
func (i *ImageForm) Next(h http.Handler) http.Handler {
	i.next = h
	return i
}
