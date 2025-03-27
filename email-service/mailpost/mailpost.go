package mailpost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	apiURL = "https://api.mailopost.ru/v1/email"
)

type Client struct {
	APIKey string
	// APISecret string
	client *http.Client
	Sender string
}

func NewClient(apiKey, sender string) *Client {
	return &Client{APIKey: apiKey, client: http.DefaultClient, Sender: sender}
}

func (c *Client) SendMessage(ctx context.Context, to, subject, body string) error {
	msg := Message{
		To:        to,
		Subject:   subject,
		Text:      body,
		HTML:      fmt.Sprintf("<p>%s</p>", body),
		FromEmail: c.Sender,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("%s/messages", apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonMsg))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("client do: %w", err)
	}
	defer resp.Body.Close()

	bodyResp, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("invalid status code: %d, body: %s", resp.StatusCode, string(bodyResp))
	}

	return nil
}

type Message struct {
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Text      string `json:"text"`
	HTML      string `json:"html"`
	FromEmail string `json:"from_email"`
}
