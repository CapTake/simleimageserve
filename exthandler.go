package main

import (
	"log"
	"net/http"
)

// Middleware interface
type Middleware interface {
	Next(http.Handler) http.Handler
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// ExtHandler adds middleware functionality to http.handler
type ExtHandler struct {
	tip http.Handler
}

// ServeHTTP if there are added middleware handlers - executes them
// otherwise processes http request
func (h *ExtHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.tip == nil {
		log.Println("ExtHandler tip is nil")
		return
	}
	h.tip.ServeHTTP(w, r)
}

// NewExtHandler wraps ordinary http.Handler in order to add
// middlewares
func NewExtHandler(h http.Handler) *ExtHandler {
	return &ExtHandler{h}
}

// Add adds http middleware in LIFO order
func (h *ExtHandler) Add(m Middleware) *ExtHandler {
	h.tip = m.Next(h.tip)
	return h
}
