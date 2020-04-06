package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type result struct {
	Status string      `json:"status"`
	Result interface{} `json:"res,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

func writeResult(w http.ResponseWriter, res interface{}) {
	w.Header().Set("Content-type", "application/json")

	encoded, err := json.Marshal(result{"OK", res, nil})
	if err != nil {
		writeError(w, err.Error(), 500)
		return
	}
	w.Write(encoded)
}

func writeError(w http.ResponseWriter, res interface{}, status int) {
	w.Header().Set("Content-type", "application/json")
	encoded, _ := json.Marshal(result{fmt.Sprintf("%v", status), nil, res})
	w.Write(encoded)
	stats.mu.Lock()
	stats.Errors++
	stats.mu.Unlock()
}
