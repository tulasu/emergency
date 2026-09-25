package calls

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

// ARI dials the student. Only system originates (AD-9).
// Stdlib only (no SIP/ARI deps per spec).
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

// Originate dials PJSIP/{to} with call_id in channel vars.
// Returns the ARI channel id so hangup targets the real leg instead of
// DELETE /channels/{call_id} which 404s and leaves the leg up (spec F).
// Empty BaseURL = dev stub: logs only, call proceeds to ringing.
func (a *ARI) Originate(to, callID string) (string, error) {
	if a == nil || a.BaseURL == "" {
		return "", nil
	}
	form := url.Values{}
	form.Set("endpoint", "PJSIP/"+to)
	form.Set("context", "trainer-out")
	form.Set("extension", "s")
	form.Set("priority", "1")
	// ponytail: Asterisk >=18 takes originate variables ONLY as a JSON body
	// object (api-docs: "the variables key in the body object"); a ?variables=
	// query pair is silently ignored and the dialplan sees an empty call_id.
	vars := map[string]string{"call_id": callID}
	// ponytail: bridge compose passes DIALOG_ADDR (dialog:9001); host-mode falls back to the dialplan default
	if addr := strings.TrimSpace(os.Getenv("DIALOG_ADDR")); addr != "" {
		vars["DIALOG_ADDR"] = addr
	}
	body, _ := json.Marshal(map[string]any{"variables": vars})
	req, err := http.NewRequest("POST", a.BaseURL+"/channels?"+form.Encode(), bytes.NewReader(body))
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
	// StasisStart/201 body carries {"id": "<channel>"}; absent → hangup
	// falls back to call_id (best-effort, tolerated 404).
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &created); err == nil {
		return created.ID, nil
	}
	return "", nil
}

// Hangup is best-effort by call_id (Stasis maps it to channel).
// ponytail: persist ARI channel id on StasisStart when events WS lands.
func (a *ARI) Hangup(callID string) error {
	if a == nil || a.BaseURL == "" {
		return nil
	}
	req, err := http.NewRequest("DELETE", a.BaseURL+"/channels/"+url.PathEscape(callID), nil)
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
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("ari hangup: %s", resp.Status)
	}
	return nil
}
