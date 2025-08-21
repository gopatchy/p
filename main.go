package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type PHandler struct {
	tmpl *template.Template
	pd   *pdClient
	mux  *http.ServeMux
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

	ph := &PHandler{
		tmpl: tmpl,
		pd:   newPDClient(routingKey),
		mux:  http.NewServeMux(),
	}

	ph.mux.HandleFunc("/{$}", ph.serveRoot)

	return ph, nil
}

func (ph *PHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ph.mux.ServeHTTP(w, r)
}

func (ph *PHandler) serveRoot(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, "Parse form: %s", err)
		return
	}

	log.Printf("%s %s %s", r.RemoteAddr, r.URL.Path, r.Form.Encode())

	m := r.Form.Get("m")
	if m == "" {
		err = ph.tmpl.Execute(w, ph.envs())
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Execute template %s: %s", ph.tmpl.Name(), err)
			return
		}

		return
	}

	err = ph.pd.sendAlert(m)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error sending to PagerDuty: %s", err)
		return
	}
	sendResponse(w, "Page sent")
}

var allowedEnvs = []string{
	"SHORT_NAME",
	"CONTACT_NAME",
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
