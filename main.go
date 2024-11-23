package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
	tmpl       *template.Template
	routingKey string
}

func NewPHandler(routingKey string) (*PHandler, error) {
	tmpl := template.New("index.html")

	tmpl.Funcs(template.FuncMap{
		"replaceAll": func(o, n, s string) string { return strings.ReplaceAll(s, o, n) },
	})

	tmpl, err := tmpl.ParseFiles("static/index.html")
	if err != nil {
		return nil, fmt.Errorf("static/index.html: %w", err)
	}

	return &PHandler{
		tmpl:       tmpl,
		routingKey: routingKey,
	}, nil
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
		err = ph.tmpl.Execute(w, ph.envs())
		if err != nil {
			http.Error(w, fmt.Sprintf("execute %s: %s\n", ph.tmpl.Name(), err), http.StatusBadRequest)
			return
		}

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

var allowedEnvs = []string{
	"CONTACT_PHONE",
	"CONTACT_SMS",
	"CONTACT_IMESSAGE",
	"CONTACT_WHATSAPP",
	"CONTACT_PAGE_EMAIL",
}

func (ph *PHandler) envs() map[string]string {
	envs := map[string]string{}

	for _, k := range allowedEnvs {
		v := os.Getenv(k)
		if v != "" {
			envs[k] = v
		}
	}

	return envs
}

func main() {
	routingKey := os.Getenv("PD_ROUTING_KEY")
	if routingKey == "" {
		log.Fatalf("please set PD_ROUTING_KEY")
	}

	ph, err := NewPHandler(routingKey)
	if err != nil {
		log.Fatalf("NewPHandler: %s", err)
	}

	http.Handle("/", ph)

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
