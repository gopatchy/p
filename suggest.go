package main

import (
	"net/http"
)

type SuggestResponse struct {
	Message string `json:"message"`
}

func sendSuggestResponse(w http.ResponseWriter, msg string) {
	sendJSON(w, SuggestResponse{Message: msg})
}
