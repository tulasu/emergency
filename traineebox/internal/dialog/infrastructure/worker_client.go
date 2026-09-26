package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var dialogHTTP = &http.Client{Timeout: 10 * time.Second}

type WorkerClient struct{}

func NewWorkerClient() *WorkerClient { return &WorkerClient{} }

func (WorkerClient) Reload(base, version, digest string, slots map[string]string, questions map[string][]string, token string) error {
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("empty dialog worker URL")
	}
	if v := os.Getenv("DIALOG_URL"); v != "" && base == "env" {
		base = v
	}
	var last error = fmt.Errorf("no attempt")
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]any{
			"version": version, "digest": digest, "slots": slots, "questions": questions,
		})
		req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(base, "/")+"/bank/reload", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("X-Service-Token", token)
		}
		resp, err := dialogHTTP.Do(req)
		if err != nil {
			last = err
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		last = fmt.Errorf("dialog reload: %s", resp.Status)
	}
	return last
}

func (WorkerClient) Lint(base string, scenario []byte, token string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{"scenario": json.RawMessage(scenario)})
	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(base, "/")+"/scenarios/lint", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Service-Token", token)
	}
	resp, err := dialogHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dialog lint: %s", resp.Status)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
