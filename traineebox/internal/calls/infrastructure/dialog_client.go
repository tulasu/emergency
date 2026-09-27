package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"traineebox/internal/calls/application"
	"traineebox/internal/calls/domain/errs"
	"traineebox/internal/calls/domain/models"
)

type DialogClient struct {
	BaseURL string
	Client  *http.Client
}

func NewDialogClient(baseURL string) *DialogClient {
	return &DialogClient{BaseURL: baseURL, Client: http.DefaultClient}
}

func (c *DialogClient) dialogBase() string {
	if c != nil && c.BaseURL != "" {
		return c.BaseURL
	}
	return os.Getenv("DIALOG_URL")
}

func (c *DialogClient) Open(ctx context.Context, call models.Call, scenarioJSON string, audio *application.AudioReference) error {
	base := strings.TrimSpace(c.dialogBase())
	if base == "" {
		return fmt.Errorf("dialog worker not configured (DIALOG_URL)")
	}
	if !json.Valid([]byte(scenarioJSON)) || strings.TrimSpace(scenarioJSON) == "" {
		return errs.ErrBadSnapshot
	}
	body := map[string]any{
		"session_id":  call.ID.String(),
		"scenario":    json.RawMessage(scenarioJSON),
		"bank_digest": call.BankDigest,
	}
	if audio != nil {
		body["audio"] = map[string]string{"ticket_id": audio.TicketID.String(), "digest": audio.Digest}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal dialog open: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/sessions/open", bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		return errs.ErrBadSnapshot
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dialog open: %s", resp.Status)
	}
	return nil
}

func (c *DialogClient) Close(ctx context.Context, sessionID string) error {
	base := strings.TrimSpace(c.dialogBase())
	if base == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"session_id": sessionID})
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/sessions/close", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
