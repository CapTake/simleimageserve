package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

// TokenParser middleware for parsing JWT token header
type TokenParser struct {
	next         http.Handler
	secretKey    []byte
	strict       bool
	errorHandler func(http.ResponseWriter, interface{}, int)
}

// NewTokenParser - initialises new token parser middleware
func NewTokenParser(secret string, strict bool, errorHandler func(http.ResponseWriter, interface{}, int)) *TokenParser {
	return &TokenParser{
		nil,
		[]byte(secret),
		strict,
		errorHandler,
	}
}

func (i *TokenParser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("x-token")
	// tokenString = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwidWlkIjoiMTA3IiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE5MjMzMzMzMzN9.yCoTzcnnojoQplpfjMzcR6-vd0QPeGJe_iHx9sSWB3k"

	if tokenString != "" {
		// claims, err := i.parseToken(tokenString)
		claims, err := i.parseToken(tokenString)
		if i.strict && err != nil {
			i.errorHandler(w, err.Error(), 400)
			return
		}
		if err == nil {
			// store claims in context
			r = r.WithContext(context.WithValue(context.Background(), userKey("auth"), claims))
		}
	}
	i.next.ServeHTTP(w, r)
}

// Next - wrap http.handler
func (i *TokenParser) Next(h http.Handler) http.Handler {
	i.next = h
	return i
}

func (i *TokenParser) parseToken(tokenString string) (jwt.MapClaims, error) {

	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return i.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// fmt.Println(claims)
		return claims, nil
	}
	return nil, errors.New("Invalid token claims")
}
