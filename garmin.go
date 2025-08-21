package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type garminClient struct {
	c      *http.Client
	apiKey string
}

type garminMessageRequest struct {
	Messages []garminMessage `json:"messages"`
}

type garminMessage struct {
	Recipients []string `json:"recipients"`
	Sender     string   `json:"sender"`
	Timestamp  string   `json:"timestamp"`
	Message    string   `json:"message"`
}

type garminMessageResponse struct {
	Count int `json:"count"`
}

func newGarminClient(apiKey string) *garminClient {
	return &garminClient{
		c:      &http.Client{},
		apiKey: apiKey,
	}
}

func (gc *garminClient) sendMessage(imei, sender, msg string) error {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(garminMessageRequest{
		Messages: []garminMessage{
			{
				Recipients: []string{imei},
				Sender:     sender,
				Timestamp:  time.Now().Format(time.RFC3339),
				Message:    msg,
			},
		},
	})

	if err != nil {
		return err
	}

	log.Printf("sending message to garmin: %s", buf.String())

	req, err := http.NewRequest("POST", "https://ipcinbound.inreachapp.com/IPC/IPCInboundApi/api/Messaging/Message", buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", gc.apiKey)

	resp, err := gc.c.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("%s", string(body))
	}

	grResp := garminMessageResponse{}
	err = json.NewDecoder(resp.Body).Decode(&grResp)
	if err != nil {
		return err
	}

	if grResp.Count != 1 {
		return fmt.Errorf("expected 1 message, got %d", grResp.Count)
	}

	return nil
}
