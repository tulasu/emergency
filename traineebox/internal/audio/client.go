package audio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	dialogmodels "traineebox/internal/dialog/domain/models"

	"github.com/google/uuid"
)

// Client is the narrow TraineeBox surface of the audio service contract.
type Client struct {
	BaseURL      string
	ServiceToken string
	HTTPClient   *http.Client
}

type EnsureRequest struct {
	TicketID       uuid.UUID         `json:"ticket_id"`
	ScenarioDigest string            `json:"scenario_digest"`
	Scenario       json.RawMessage   `json:"scenario"`
	Slots          map[string]string `json:"slots"`
	UrgeSlots      map[string]string `json:"urge_slots"`
}

type EnsureResult struct {
	StatusCode int
	Status     string
}

type CanonicalScenario struct {
	JSON   json.RawMessage
	Digest string
	Facts  int
}

func Canonicalize(raw []byte, knownSlots map[string]bool) (CanonicalScenario, error) {
	scenario, err := dialogmodels.Validate(raw, knownSlots)
	if err != nil {
		return CanonicalScenario{}, err
	}
	canonical, err := json.Marshal(scenario)
	if err != nil {
		return CanonicalScenario{}, fmt.Errorf("marshal canonical scenario: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return CanonicalScenario{
		JSON: canonical, Digest: hex.EncodeToString(digest[:]), Facts: len(scenario.Facts),
	}, nil
}

func (c Client) Configured() bool {
	return strings.TrimSpace(c.BaseURL) != ""
}

func (c Client) Ensure(ctx context.Context, body EnsureRequest) (EnsureResult, error) {
	payload, statusCode, err := c.do(ctx, http.MethodPost, "/v1/ensure", body)
	if err != nil {
		return EnsureResult{}, err
	}
	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return EnsureResult{}, fmt.Errorf("decode audio ensure response: %w", err)
	}
	if response.Status == "" {
		return EnsureResult{}, fmt.Errorf("decode audio ensure response: status required")
	}
	return EnsureResult{StatusCode: statusCode, Status: response.Status}, nil
}

func (c Client) EnsureWithRetry(ctx context.Context, body EnsureRequest) (EnsureResult, error) {
	var lastErr error
	for range 3 {
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		result, err := c.Ensure(attemptCtx, body)
		cancel()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	return EnsureResult{}, fmt.Errorf("ensure audio after 3 attempts: %w", lastErr)
}

func (c Client) DeleteTicket(ctx context.Context, ticketID uuid.UUID) error {
	_, _, err := c.do(ctx, http.MethodDelete, "/v1/tickets/"+ticketID.String(), nil)
	return err
}

func (c Client) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	base := strings.TrimSuffix(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, 0, fmt.Errorf("audio service not configured (AUDIO_URL)")
	}
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, payload)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Service-Token", c.ServiceToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, fmt.Errorf("audio %s %s: %s", method, path, resp.Status)
	}
	return response, resp.StatusCode, nil
}
