package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type PDAlert struct {
	RoutingKey  string    `json:"routing_key"`
	EventAction string    `json:"event_action"`
	Payload     PDPayload `json:"payload"`
}

type PDPayload struct {
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
}

type PHandler struct {
	routingKey string
	next http.Handler
}

func NewPHandler(routingKey string, next http.Handler) *PHandler {
	return &PHandler{
		routingKey: routingKey,
		next: next,
	}
}

func (ph *PHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid form: %s\n", err), http.StatusBadRequest)
		return
	}

	log.Printf("%s %s", r.RemoteAddr, r.Form.Encode())

	msg := r.Form.Get("msg")
	if msg == "" {
		ph.next.ServeHTTP(w, r)
		return
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(PDAlert{
		RoutingKey:  ph.routingKey,
		EventAction: "trigger",
		Payload: PDPayload{
			Summary:  msg,
			Source:   r.RemoteAddr,
			Severity: "critical",
		},
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create PD request: %s\n", err), http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest("POST", "https://events.pagerduty.com/v2/enqueue", buf)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create HTTP request: %s\n", err), http.StatusBadRequest)
		return
	}

	c := &http.Client{}
	res, err := c.Do(req)

	if err != nil {
		http.Error(w, fmt.Sprintf("error from PD: %s\n", err), http.StatusBadRequest)
    	return
	}

	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != 202 {
		http.Error(w, fmt.Sprintf("error from PD: %s", string(body)), http.StatusBadRequest)
    	return
	}
	
	w.Write([]byte("page sent\n"))
}

func main() {
	routingKey := os.Getenv("PD_ROUTING_KEY")
	if routingKey == "" {
		log.Fatalf("please set PD_ROUTING_KEY")
	}

	http.Handle("/", NewPHandler(routingKey, http.FileServer(http.Dir("./static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	bind := fmt.Sprintf(":%s", port)
	log.Printf("listening on %s", bind)

	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Fatalf("listen: %s", err)
	}
}
