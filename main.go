package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/openai/openai-go"
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
	ph.mux.HandleFunc("/suggest", ph.serveSuggest)

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
		sendError(w, http.StatusInternalServerError, "Send PD alert: %s", err)
		return
	}
	sendResponse(w, "Page sent")
}

func (ph *PHandler) serveSuggest(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		sendError(w, http.StatusBadRequest, "Parse form: %s", err)
		return
	}

	log.Printf("%s %s %s", r.RemoteAddr, r.URL.Path, r.Form.Encode())

	m := r.Form.Get("m")
	if m == "" {
		sendError(w, http.StatusBadRequest, "m= param required")
		return
	}

	c := openai.NewClient()

	comp, err := c.Chat.Completions.New(r.Context(), openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(`You are an assistant helping users to write good text to include in an urgent page sent to a person. Good page text contains a very brief description of the problem (e.g. "down" or "slow"), the systems it affects (acronyms for system names are fine), the identity of the sender (first names are fine), and how to contact them (e.g. a phone number or incident Slack channel). The request will consist of just the user's proposed page text. Respond with just a very brief message suggesting improvements that the sender might make or saying "Looks good -- send it!". Remember that the user is likely in an urgent, stressful situation, so make your response brief and err on the side of assuming that the message is sufficient if the text might be OK. Assume that the recipient already knows the message is urgent so the sender doesn't have to specify urgency.`),

			openai.UserMessage(m),
		}),
		Model: openai.F(openai.ChatModelGPT4o),
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "OpenAI error: %s", err)
		return
	}

	sendSuggestResponse(w, comp.Choices[0].Message.Content)
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
