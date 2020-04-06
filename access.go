package main

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

func isAccessAllowed(r *http.Request) bool {
	tokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwidWlkIjoiMTA3IiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE5MjMzMzMzMzN9.yCoTzcnnojoQplpfjMzcR6-vd0QPeGJe_iHx9sSWB3k"
	// token, ok := r.Header["X-Token"]
	// if !ok {
	// 	return false
	// }
	// return token != nil
	_, err := parseToken(tokenString)
	return err == nil
}

func parseToken(tokenString string) (interface{}, error) {
	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte("266bitsecret"), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Println(claims["uid"], token)
	} else {
		fmt.Println(err)
	}
	return token, err
}
