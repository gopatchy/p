package main

import (
	"fmt"
	"net/http"
)

type Response struct {
	Message string `json:"message"`
}

func sendResponse(w http.ResponseWriter, msg string, args ...any) {
	sendJSON(w, Response{Message: fmt.Sprintf(msg, args...)})
}
