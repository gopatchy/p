package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
		return fmt.Errorf("Encode PD alert: %s", err)
	}

	req, err := http.NewRequest("POST", "https://events.pagerduty.com/v2/enqueue", buf)
	if err != nil {
		return fmt.Errorf("Create HTTP request: %s", err)
	}

	resp, err := pd.c.Do(req)
	if err != nil {
		return fmt.Errorf("Call PagerDuty: %s", err)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 202 {
		return fmt.Errorf("PagerDuty error: %s", string(body))
	}

	return nil
}
