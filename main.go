package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello! you've requested %s\n", r.URL.Path)
	})

	http.Handle("/static", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	bindAddr := fmt.Sprintf(":%s", port)
	fmt.Printf("==> Server listening at %s 🚀\n", bindAddr)

	if err := http.ListenAndServe(bindAddr, nil); err != nil {
		panic(err)
	}
}
