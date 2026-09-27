// Package services implements audio synthesis and queue processing.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SynthClient calls one audio-synth daemon.
type SynthClient struct {
	Base string // e.g. http://audio-synth:8003, no trailing slash
	HTTP *http.Client
}

// NewSynthClient builds a client with per-call timeout.
func NewSynthClient(base string, timeout time.Duration) *SynthClient {
	return &SynthClient{
		Base: strings.TrimSuffix(base, "/"),
		HTTP: &http.Client{Timeout: timeout},
	}
}

type synthReq struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
}

// Synth synthesizes text and returns wav bytes (8k mono s16).
// The daemon is singleflight-serialized internally (one _lock per process).
func (c *SynthClient) Synth(ctx context.Context, text, voice string) ([]byte, error) {
	body, _ := json.Marshal(synthReq{Text: text, Voice: voice})
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/synth", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		peek, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("synth: %s: %s", resp.Status, strings.TrimSpace(string(peek)))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// Check pings GET /health; non-200 is an error.
func (c *SynthClient) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.Base+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != 200 {
		return fmt.Errorf("synth health: %s", resp.Status)
	}
	return nil
}
