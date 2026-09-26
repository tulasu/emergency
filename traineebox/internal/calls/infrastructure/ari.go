package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ARI struct {
	BaseURL  string
	User     string
	Password string
	Client   *http.Client
}

func ARIFromEnv() *ARI {
	return &ARI{
		BaseURL:  strings.TrimSuffix(os.Getenv("ARI_URL"), "/"),
		User:     os.Getenv("ARI_USER"),
		Password: os.Getenv("ARI_PASS"),
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *ARI) Originate(to, callID string) (string, error) {
	if a == nil || a.BaseURL == "" {
		return "", nil
	}
	form := url.Values{}
	form.Set("endpoint", "PJSIP/"+to)
	form.Set("context", "trainer-out")
	form.Set("extension", "s")
	form.Set("priority", "1")
	vars := map[string]string{"call_id": callID}
	if addr := strings.TrimSpace(os.Getenv("DIALOG_ADDR")); addr != "" {
		vars["DIALOG_ADDR"] = addr
	}
	body, _ := json.Marshal(map[string]any{"variables": vars})
	req, err := http.NewRequest(http.MethodPost, a.BaseURL+"/channels?"+form.Encode(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.User != "" {
		req.SetBasicAuth(a.User, a.Password)
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ari originate: %s", resp.Status)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &created); err == nil {
		return created.ID, nil
	}
	return "", nil
}

func (a *ARI) Hangup(callID string) error {
	if a == nil || a.BaseURL == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodDelete, a.BaseURL+"/channels/"+url.PathEscape(callID), nil)
	if err != nil {
		return err
	}
	if a.User != "" {
		req.SetBasicAuth(a.User, a.Password)
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("ari hangup: %s", resp.Status)
	}
	return nil
}
