package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type pdAlert struct {
	RoutingKey  string    `json:"routing_key"`
	EventAction string    `json:"event_action"`
	Payload     pdPayload `json:"payload"`
}

type pdPayload struct {
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
}

type pdResponse struct {
	DedupKey string `json:"dedup_key"`
	Message  string `json:"message"`
	Status   string `json:"status"`
}

type pdClient struct {
	c          *http.Client
	routingKey string
}

func newPDClient(routingKey string) *pdClient {
	return &pdClient{
		c:          &http.Client{},
		routingKey: routingKey,
	}
}

func (pd *pdClient) sendAlert(msg string) error {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(pdAlert{
		RoutingKey:  pd.routingKey,
		EventAction: "trigger",
		Payload: pdPayload{
			Summary:  msg,
			Source:   "p",
			Severity: "critical",
		},
	})

	if err != nil {
		return err
	}

	log.Printf("[->pagerduty] %s", buf.String())

	req, err := http.NewRequest("POST", "https://events.pagerduty.com/v2/enqueue", buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := pd.c.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 202 {
		return fmt.Errorf("%s", string(body))
	}

	log.Printf("[<-pagerduty] %s", string(body))

	pdResp := pdResponse{}
	err = json.Unmarshal(body, &pdResp)
	if err != nil {
		return err
	}

	if pdResp.Status != "success" {
		return fmt.Errorf("%s: %s", pdResp.Status, pdResp.Message)
	}

	return nil
}
