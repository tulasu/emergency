package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"traineebox/internal/generation/domain/models"
)

type TicketgenClient struct {
	BaseURL string
	Client  *http.Client
}

func NewTicketgenClient(baseURL string) *TicketgenClient {
	return &TicketgenClient{
		BaseURL: strings.TrimSpace(baseURL),
		Client:  http.DefaultClient,
	}
}

func (c *TicketgenClient) Draft(ctx context.Context, prompt string) (string, string, models.DraftReference, error) {
	body, _ := json.Marshal(map[string]string{"prompt": prompt})
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(c.BaseURL, "/")+"/draft", bytes.NewReader(body))
	if err != nil {
		return "", "", models.DraftReference{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", models.DraftReference{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", "", models.DraftReference{}, fmt.Errorf("ticketgen draft: %s", resp.Status)
	}
	var out struct {
		DraftTitle     string                `json:"draft_title"`
		ScenarioText   string                `json:"scenario_text"`
		DraftReference models.DraftReference `json:"draft_reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", models.DraftReference{}, err
	}
	if strings.TrimSpace(out.DraftTitle) == "" || strings.TrimSpace(out.ScenarioText) == "" {
		return "", "", models.DraftReference{}, fmt.Errorf("ticketgen returned empty draft")
	}
	return out.DraftTitle, out.ScenarioText, out.DraftReference, nil
}
