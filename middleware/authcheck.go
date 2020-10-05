package middleware

import (
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

type userKey string

type authMap map[string]interface{}

// AuthCheck middleware for parsing JWT token header
type AuthCheck struct {
	next         http.Handler
	ErrorHandler func(http.ResponseWriter, interface{}, int)
}

func (i *AuthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	v, ok := r.Context().Value(userKey("auth")).(jwt.MapClaims)
	if v == nil || !ok {
		i.ErrorHandler(w, "Authentication required", 401)
		return
	}
	log.Printf("%T: %v", v, v)
	// TODO: fix this! mapping doesn't work
	// if v["uid"].(string) == "" {
	// 	i.ErrorHandler(w, "Authentification failed", 403)
	// 	return
	// }

	i.next.ServeHTTP(w, r)
}

// Next - wrap http.handler
func (i *AuthCheck) Next(h http.Handler) http.Handler {
	i.next = h
	return i
}
