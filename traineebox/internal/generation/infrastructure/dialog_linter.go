package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DialogLinter struct {
	URLs         []string
	ServiceToken string
	Client       *http.Client
}

func NewDialogLinter(urls []string, serviceToken string) *DialogLinter {
	return &DialogLinter{
		URLs:         urls,
		ServiceToken: serviceToken,
		Client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (l *DialogLinter) Lint(ctx context.Context, snapshot string) (unreachable []string, skipped bool, err error) {
	if len(l.URLs) == 0 {
		return nil, true, nil
	}
	var last error
	for _, base := range l.URLs {
		out, lintErr := l.postLint(ctx, base, snapshot)
		if lintErr != nil {
			last = lintErr
			continue
		}
		if bad, _ := out["unreachable"].([]any); len(bad) > 0 {
			codes := make([]string, 0, len(bad))
			for _, v := range bad {
				codes = append(codes, fmt.Sprint(v))
			}
			return codes, false, nil
		}
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("dialog lint unavailable: %v", last)
}

func (l *DialogLinter) postLint(ctx context.Context, base, snapshot string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{"scenario": json.RawMessage(snapshot)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(base, "/")+"/scenarios/lint", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.ServiceToken != "" {
		req.Header.Set("X-Service-Token", l.ServiceToken)
	}
	resp, err := l.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dialog lint: %s", resp.Status)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
