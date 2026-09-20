// Шаг 5. Go-оркестратор: HTTP POST /calls -> ARI originate -> RTP-помпа.
// SIP/RTP-стек держит Asterisk (шаг 3), понимание — Python (шаг 6).
// Go только: реестр call_id, originate, UDP-сокет на звонок, прокси байт
// в Python и обратно. stdlib only. Stateless: рестарт роняет активные.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type Config struct {
	ARIURL  string // http://asterisk:8088/ari
	ARIUser string
	ARIPass string
	PyURL   string // http://python:8000
	RTPBase int    // первый UDP-порт, дальше +1 на звонок
}

func cfgFromEnv() Config {
	base := 10000
	fmt.Sscanf(os.Getenv("RTP_BASE"), "%d", &base)
	return Config{
		ARIURL:  env("ARI_URL", "http://127.0.0.1:8088/ari"),
		ARIUser: env("ARI_USER", "admin"),
		ARIPass: env("ARI_PASS", "admin"),
		PyURL:   env("PY_URL", "http://127.0.0.1:8000"),
		RTPBase: base,
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Call — всё состояние звонка. Сценарий Python грузит по scenarioID
// при Open, журнал хранит Python; тут только транспорт.
type Call struct {
	ID         string `json:"call_id"`
	To         string `json:"to"`
	ScenarioID string `json:"scenario_id"`
	RTPPort    int    `json:"rtp_port"`

	conn   *net.UDPConn
	cancel chan struct{}
}

type Hub struct {
	mu    sync.Mutex
	calls map[string]*Call
	next  int
	cfg   Config
}

func newCallID() string {
	var b [16]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// originate просит Asterisk позвонить. Переменные scenario_id/call_id едут
// в dialplan trainer-out (шаг 3), RTP-порт — для UnicastRTP-канала назад.
func (h *Hub) originate(to, scenario, callID string, port int) error {
	form := url.Values{}
	form.Set("endpoint", "PJSIP/"+to)
	form.Set("context", "trainer-out")
	form.Set("extension", "s")
	form.Set("priority", "1")
	form.Set("variables",
		fmt.Sprintf("scenario_id=%s,call_id=%s,go_rtp_port=%d", scenario, callID, port))
	req, _ := http.NewRequest("POST", h.cfg.ARIURL+"/channels?"+form.Encode(), nil)
	req.SetBasicAuth(h.cfg.ARIUser, h.cfg.ARIPass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ari originate: %s", resp.Status)
	}
	return nil
}

func (h *Hub) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		To         string `json:"to"`
		ScenarioID string `json:"scenario_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.To == "" {
		http.Error(w, `{"error":"need to, scenario_id"}`, 400)
		return
	}
	// сценарий проверяем сразу об Python — 404 раньше, чем зазвонит
	if err := pyOpenCheck(h.cfg.PyURL, in.ScenarioID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 404)
		return
	}
	h.mu.Lock()
	port := h.cfg.RTPBase + h.next
	h.next++
	c := &Call{ID: newCallID(), To: in.To, ScenarioID: in.ScenarioID,
		RTPPort: port, cancel: make(chan struct{})}
	h.calls[c.ID] = c
	h.mu.Unlock()

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err == nil {
		c.conn, err = net.ListenUDP("udp", addr)
	}
	if err != nil {
		h.drop(c.ID)
		http.Error(w, `{"error":"no rtp port"}`, 500)
		return
	}
	if err := h.originate(in.To, in.ScenarioID, c.ID, port); err != nil {
		h.drop(c.ID)
		http.Error(w, `{"error":"originate failed"}`, 502)
		return
	}
	go h.pump(c) // RTP <-> Python, пока висит канал
	w.WriteHeader(202)
	json.NewEncoder(w).Encode(c)
}

// pump гоняет RTP туда-обратно. Разбор SLIN/VAD/STT — в Python Call (шаг 4);
// тут байты как есть: Asterisk шлёт alaw, Python просит PCM — конверсия
// на стороне Python, чтобы Go остался без кодеков.
func (h *Hub) pump(c *Call) {
	buf := make([]byte, 2048)
	for {
		select {
		case <-c.cancel:
			return
		default:
		}
		c.conn.SetReadDeadline(deadline())
		n, peer, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		// эхо-помпа v1: складываем в Python-сессию через /rtp, ответ
		// возвращаем тому же peer. Полный аудиопуть — шаг 6.
		out := pyRTP(h.cfg.PyURL, c.ID, c.ScenarioID, buf[:n])
		if len(out) > 0 && peer != nil {
			c.conn.WriteToUDP(out, peer)
		}
	}
}

func (h *Hub) drop(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.calls[id]; ok {
		close(c.cancel)
		if c.conn != nil {
			c.conn.Close()
		}
		delete(h.calls, id)
	}
}

func (h *Hub) handleList(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	list := make([]*Call, 0, len(h.calls))
	for _, c := range h.calls {
		list = append(list, c)
	}
	h.mu.Unlock()
	json.NewEncoder(w).Encode(list)
}

func (h *Hub) handleHangup(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/calls/")
	h.drop(id)
	// сам канал роняет Asterisk (BYE); ARI-delete — когда понадобится
	w.WriteHeader(204)
}

func main() {
	h := &Hub{calls: map[string]*Call{}, cfg: cfgFromEnv()}
	http.HandleFunc("POST /calls", h.handleCreate)
	http.HandleFunc("GET /calls", h.handleList)
	http.HandleFunc("DELETE /calls/", h.handleHangup)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
