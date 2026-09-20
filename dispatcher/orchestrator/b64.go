// Мост к Python-серверу (шаг 6). JSON поверх HTTP, без gRPC-зависимостей:
// контракты /sessions/open и /rtp зеркалят Service.open и media.Call.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func unb64(s string) []byte {
	b, _ := base64.StdEncoding.DecodeString(s)
	return b
}

func deadline() time.Time { return time.Now().Add(2 * time.Second) }

// pyOpenCheck проверяет сценарий до originate: 404 — звонить не будем.
func pyOpenCheck(pyURL, scenario string) error {
	body, _ := json.Marshal(map[string]string{"scenario_id": scenario})
	resp, err := http.Post(pyURL+"/sessions/check", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("python unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return fmt.Errorf("unknown scenario %s", scenario)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("python status %s", resp.Status)
	}
	return nil
}

// pyRTP отдаёт входящий чанк в Python-сессию звонка, возвращает исходящий.
// Пустой ответ — молчим в канал. Ошибка — тоже молчим (не рвём звонок).
func pyRTP(pyURL, callID, scenario string, chunk []byte) []byte {
	body, _ := json.Marshal(map[string]string{
		"session_id": callID, "scenario_id": scenario,
		"audio_b64": b64(chunk),
	})
	resp, err := http.Post(pyURL+"/rtp", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode >= 300 {
		if err == nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	var out struct {
		AudioB64 string `json:"audio_b64"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.AudioB64 == "" {
		return nil
	}
	return unb64(out.AudioB64)
}
