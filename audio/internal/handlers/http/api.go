// Package httpapi serves the audio HTTP surface (spec 06):
// stdlib net/http ServeMux, hand-encoded JSON, no huma/chi — six handlers.
// Mutating routes + blob reads require X-Service-Token (shared
// INTERNAL_SERVICE_TOKEN); manifest/health/stats stay open for dialog.
package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"audio/internal/config"
	"audio/internal/domain"
	"audio/internal/repositories"
	"audio/internal/services"
	"audio/pkg/hash"
	"audio/pkg/s3"

	"github.com/google/uuid"
)

// API wires HTTP handlers to repositories/S3/stats.
type API struct {
	cfg   config.Config
	store *repositories.Store
	s3    *s3.Client
	stats *services.Stats
}

// New builds the API.
func New(cfg config.Config, st *repositories.Store, s3c *s3.Client, stt *services.Stats) *API {
	return &API{cfg: cfg, store: st, s3: s3c, stats: stt}
}

// Register mounts the six spec routes plus the root probe.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/ensure", a.requireToken(a.ensure))
	mux.HandleFunc("GET /v1/tickets/{id}/manifest", a.getManifest)
	mux.HandleFunc("GET /v1/blobs/{hash}", a.requireToken(a.getBlob))
	mux.HandleFunc("DELETE /v1/tickets/{id}", a.requireToken(a.deleteTicket))
	mux.HandleFunc("POST /v1/sweep/run", a.requireToken(a.runSweep))
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /v1/stats", a.statsHandler)
	mux.HandleFunc("GET /", a.health)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// requireToken enforces X-Service-Token; empty configured token fails closed.
func (a *API) requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Service-Token")
		if a.cfg.InternalToken == "" || got == "" || got != a.cfg.InternalToken {
			writeErr(w, http.StatusUnauthorized, "bad service token")
			return
		}
		next(w, r)
	}
}

type ensureReq struct {
	TicketID       string            `json:"ticket_id"`
	ScenarioDigest string            `json:"scenario_digest"`
	Scenario       json.RawMessage   `json:"scenario"`
	Slots          map[string]string `json:"slots"`
	UrgeSlots      map[string]string `json:"urge_slots"`
	Voice          string            `json:"voice"`
}

// POST /v1/ensure — idempotent: digest match returns stored status + missing
// with no writes; new digest diffs by texthash (1-fact edit = 1-2 missing).
func (a *API) ensure(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad body")
		return
	}
	var in ensureReq
	if err := json.Unmarshal(body, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	ticketID, err := uuid.Parse(in.TicketID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad ticket_id")
		return
	}
	if strings.TrimSpace(in.ScenarioDigest) == "" {
		writeErr(w, http.StatusBadRequest, "scenario_digest required")
		return
	}
	voice := in.Voice
	if voice == "" {
		voice = a.cfg.Voice
	}
	if !json.Valid(in.Scenario) {
		writeErr(w, http.StatusBadRequest, "bad scenario json")
		return
	}
	known := map[string]bool{}
	for k := range in.Slots {
		known[k] = true
	}
	sc, err := domain.Validate(in.Scenario, known)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	frags := domain.Enumerate(sc, in.Slots, in.UrgeSlots, renderFixture.Generic, renderFixture.Slowdown, renderFixture.Urge)
	sums := make([][32]byte, len(frags))
	for i, f := range frags {
		sums[i] = hash.Texthash(hash.Normalize(f.Text), voice, a.cfg.Rate, hash.Model)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	status, missing, matched, err := a.store.Ensure(ctx, ticketID, in.ScenarioDigest, voice, frags, sums)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ensure failed")
		return
	}
	code := http.StatusAccepted
	if matched && status == "ready" {
		code = http.StatusOK
	}
	writeJSON(w, code, map[string]any{
		"scenario_digest": in.ScenarioDigest,
		"status":          status,
		"missing":         missing,
	})
}

// GET /v1/tickets/{id}/manifest — read-only cache for dialog open.
func (a *API) getManifest(w http.ResponseWriter, r *http.Request) {
	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad ticket id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	m, err := a.store.GetManifest(ctx, ticketID)
	if errors.Is(err, repositories.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no manifest")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "manifest failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scenario_digest": m.Digest,
		"voice":           m.Voice,
		"status":          m.Status,
		"fragments":       m.Fragments,
		"error":           m.Error,
	})
}

// GET /v1/blobs/{hash} — proxy from RustFS (v1), ETag=hex, immutable.
func (a *API) getBlob(w http.ResponseWriter, r *http.Request) {
	sum, err := hash.Decode(strings.ToLower(r.PathValue("hash")))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad hash")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	voice, rate, err := a.store.BlobVoice(ctx, sum)
	if errors.Is(err, repositories.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no blob")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "blob lookup failed")
		return
	}
	body, err := a.s3.Get(ctx, hash.S3Key(voice, rate, hash.Hex(sum)))
	if err != nil {
		writeErr(w, http.StatusBadGateway, "blob fetch failed")
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "immutable")
	w.Header().Set("ETag", `"`+hash.Hex(sum)+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// DELETE /v1/tickets/{id} — drops refs + manifest; blobs go orphan to sweep.
func (a *API) deleteTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad ticket id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	n, err := a.store.DeleteTicket(ctx, ticketID)
	if errors.Is(err, repositories.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no ticket")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refs_dropped": n})
}

// POST /v1/sweep/run?limit=500 — service-token manual sweep on top of cron.
func (a *API) runSweep(w http.ResponseWriter, r *http.Request) {
	limit := a.cfg.SweepLimit
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		var n int
		if _, err := hex.DecodeString(""); err == nil {
			_ = n
		}
		if parsed, perr := parseLimit(q); perr == nil && parsed > 0 {
			limit = parsed
		}
	}
	dels, bytes, err := SweepOnce(r.Context(), a.store, a.s3, sweepTTL(), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "sweep failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted_blobs": dels,
		"deleted_bytes": bytes,
	})
}

// GET /health — {ok, queue_depth, blobs, refs}.
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	q, b, rf, err := a.store.Counts(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"queue_depth": q,
		"blobs":       b,
		"refs":        rf,
	})
}

// GET /v1/stats — synth p50/p95 + queue depth; hit/miss counted by dialog.
func (a *API) statsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	q, _, _, _ := a.store.Counts(ctx)
	p50, p95 := a.stats.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"synth_ms_p50": p50,
		"synth_ms_p95": p95,
		"queue_depth":  q,
	})
}
