// Package services implements audio synthesis and queue processing.
package services

import (
	"audio/internal/config"
	"audio/internal/repositories"
	"audio/pkg/hash"
	"audio/pkg/s3"
	"audio/pkg/wav"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"
)

// piiDigits masks phone-like runs so texts never land in logs unredacted
// (same redact as traineebox generation worker.go).
var piiDigits = regexp.MustCompile(`\d{7,}`)

func redact(s string) string {
	return piiDigits.ReplaceAllString(s, "***")
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// Worker drives synth_queue → audio-synth → S3 → blobs/refs/manifest.
type Worker struct {
	cfg      config.Config
	store    *repositories.Store
	s3       *s3.Client
	synth    *SynthClient
	stats    *Stats
	callback *readyCallback
}

// NewWorker builds the queue worker.
func NewWorker(cfg config.Config, st *repositories.Store, s3c *s3.Client, sc *SynthClient, stt *Stats) *Worker {
	return &Worker{
		cfg:      cfg,
		store:    st,
		s3:       s3c,
		synth:    sc,
		stats:    stt,
		callback: newReadyCallback(cfg),
	}
}

// Loop claims jobs until ctx stops. Paused only in offline/CPU mode
// (spec 06: orchestrator off under load); sweeps keep running separately.
func (w *Worker) Loop(ctx context.Context) {
	if w.cfg.PauseWorker {
		log.Print("audio worker paused (AUDIO_WORKER_PAUSED=1)")
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		drove, err := w.DriveOnce(ctx)
		if err != nil {
			log.Printf("audio worker: %v", redact(err.Error()))
		}
		// TODO: replace idle polling with a wakeup on new jobs, while retaining
		// recovery of expired claims.
		if !drove {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.cfg.WorkerPollInterval):
			}
		}
	}
}

// DriveOnce claims one job and synthesizes it. No work → (false, nil).
func (w *Worker) DriveOnce(ctx context.Context) (bool, error) {
	items, err := w.store.Claim(ctx, 1)
	if err != nil {
		return true, err
	}
	if len(items) == 0 {
		return false, nil
	}
	for _, it := range items {
		start := time.Now()
		sctx, cancel := context.WithTimeout(ctx, w.cfg.WorkerSynthTimeout)
		wavBytes, err := w.synth.Synth(sctx, it.Text, it.Voice)
		cancel()
		if err != nil {
			_ = w.store.FailItem(ctx, it, "synth failed: "+trunc(redact(err.Error()), 200))
			continue
		}
		info, err := wav.Check(wavBytes)
		if err != nil {
			_ = w.store.FailItem(ctx, it, "bad wav: "+trunc(redact(err.Error()), 200))
			continue
		}
		key := hash.S3Key(it.Voice, w.cfg.Rate, hash.Hex(it.Hash))
		if err := w.s3.Put(ctx, key, wavBytes); err != nil {
			_ = w.store.FailItem(ctx, it, "s3 put failed: "+trunc(redact(err.Error()), 200))
			continue
		}
		if err := w.store.FinishItem(ctx, it, w.cfg.Rate, info.Bytes, info.DurS); err != nil {
			if errors.Is(err, repositories.ErrClaimLost) {
				continue
			}
			_ = w.store.FailItem(ctx, it, "db finish failed: "+trunc(redact(err.Error()), 200))
			continue
		}
		w.stats.Observe(float64(time.Since(start).Milliseconds()))
		if err := w.notifyReady(ctx, it.Hash); err != nil {
			log.Printf("audio worker ready callback: %v", redact(err.Error()))
		}
	}
	return true, nil
}

const (
	readyCallbackAttempts   = 3
	readyCallbackTimeout    = 5 * time.Second
	readyCallbackRetryDelay = 100 * time.Millisecond
)

type readyCallback struct {
	baseURL    string
	token      string
	httpClient *http.Client
	retryDelay time.Duration
}

type readyCallbackRequest struct {
	TicketID       string `json:"ticket_id"`
	ScenarioDigest string `json:"scenario_digest"`
}

func newReadyCallback(cfg config.Config) *readyCallback {
	return &readyCallback{
		baseURL:    cfg.TraineeBoxURL,
		token:      cfg.InternalToken,
		httpClient: &http.Client{Timeout: readyCallbackTimeout},
		retryDelay: readyCallbackRetryDelay,
	}
}

// notifyReady reads all ready manifests satisfied by a committed synthesis
// hash. The notification is intentionally best-effort: a TraineeBox outage
// must not requeue audio.
func (w *Worker) notifyReady(ctx context.Context, sum [32]byte) error {
	if w.callback == nil || w.callback.baseURL == "" {
		return nil
	}
	manifests, err := w.store.ReadyManifestsForHash(ctx, sum)
	if err != nil {
		return fmt.Errorf("load ready manifests: %w", err)
	}
	var callbackErr error
	for _, manifest := range manifests {
		if err := w.callback.notify(ctx, manifest); err != nil {
			callbackErr = errors.Join(callbackErr, err)
		}
	}
	return callbackErr
}

// notify posts only final ready manifests. Retriable transport and HTTP
// failures are bounded and cannot affect completed synthesis work.
func (c *readyCallback) notify(ctx context.Context, manifest repositories.Manifest) error {
	if c.baseURL == "" || manifest.Status != "ready" {
		return nil
	}
	payload, err := json.Marshal(readyCallbackRequest{
		TicketID:       manifest.TicketID.String(),
		ScenarioDigest: manifest.Digest,
	})
	if err != nil {
		return fmt.Errorf("encode callback: %w", err)
	}

	var lastErr error
	for attempt := range readyCallbackAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			c.baseURL+"/internal/audio/ready", bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create callback request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Service-Token", c.token)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("post callback: %w", err)
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
		resp.Body.Close()
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return nil
		}
		lastErr = fmt.Errorf("callback returned %s", resp.Status)
		if !retryableCallbackStatus(resp.StatusCode) {
			return lastErr
		}
	}
	return lastErr
}

func retryableCallbackStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

// Sweeper deletes orphan blobs past TTL every interval (plus manual
// POST /v1/sweep/run). S3 keys first, then DB rows — crash-safe order.
type Sweeper struct {
	cfg   config.Config
	store *repositories.Store
	s3    *s3.Client
}

// NewSweeper builds the GC loop.
func NewSweeper(cfg config.Config, st *repositories.Store, s3c *s3.Client) *Sweeper {
	return &Sweeper{cfg: cfg, store: st, s3: s3c}
}

// Loop runs sweeps until ctx stops.
func (s *Sweeper) Loop(ctx context.Context) {
	t := time.NewTicker(s.cfg.SweepInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.Once(ctx); err != nil {
				log.Printf("audio sweep: %v", redact(err.Error()))
			}
		}
	}
}

// Once runs one bounded sweep pass, keeping each blob locked through S3 deletion.
func (s *Sweeper) Once(ctx context.Context) error {
	n, bytes, err := s.store.SweepOrphans(ctx, 7*24*time.Hour, s.cfg.SweepLimit,
		func(ctx context.Context, o repositories.Orphan) error {
			return s.s3.Delete(ctx, hash.S3Key(o.Voice, o.Rate, hash.Hex(o.Hash)))
		})
	if err != nil {
		return err
	}
	if n > 0 {
		log.Printf("audio sweep: deleted_blobs=%d deleted_bytes=%d", n, bytes)
	}
	return nil
}
