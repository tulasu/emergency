// Package worker owns the synth queue and the sweep cron (spec 06).
// Loop shape mirrors traineebox generation worker: claim batch via
// FOR UPDATE SKIP LOCKED, idle poll, redact PII in errors. Singleflight
// per hash comes free from the synth_queue PK.
package worker

import (
	"context"
	"log"
	"regexp"
	"time"

	"audio/internal/config"
	"audio/internal/hash"
	"audio/internal/s3"
	"audio/internal/stats"
	"audio/internal/store"
	"audio/internal/synth"
	"audio/internal/wav"
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
	cfg   config.Config
	store *store.Store
	s3    *s3.Client
	synth *synth.Client
	stats *stats.Stats
}

// New builds the queue worker.
func New(cfg config.Config, st *store.Store, s3c *s3.Client, sc *synth.Client, stt *stats.Stats) *Worker {
	return &Worker{cfg: cfg, store: st, s3: s3c, synth: sc, stats: stt}
}

// Loop claims batches until ctx stops. Paused only in offline/CPU mode
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
		if !drove {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.cfg.WorkerPollInterval):
			}
		}
	}
}

// DriveOnce claims one batch and synthesizes it. No work → (false, nil).
func (w *Worker) DriveOnce(ctx context.Context) (bool, error) {
	items, err := w.store.Claim(ctx, w.cfg.WorkerBatch)
	if err != nil {
		return true, err
	}
	if len(items) == 0 {
		return false, nil
	}
	for _, it := range items {
		start := time.Now()
		sctx, cancel := context.WithTimeout(ctx, w.cfg.WorkerSynthTimeout)
		wavBytes, err := w.synth.Synth(sctx, it.Text, w.cfg.Voice)
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
		key := hash.S3Key(w.cfg.Voice, w.cfg.Rate, hash.Hex(it.Hash))
		if err := w.s3.Put(ctx, key, wavBytes); err != nil {
			_ = w.store.FailItem(ctx, it, "s3 put failed: "+trunc(redact(err.Error()), 200))
			continue
		}
		if err := w.store.FinishItem(ctx, it, w.cfg.Voice, w.cfg.Rate, info.Bytes, info.DurS); err != nil {
			_ = w.store.FailItem(ctx, it, "db finish failed: "+trunc(redact(err.Error()), 200))
			continue
		}
		w.stats.Observe(float64(time.Since(start).Milliseconds()))
	}
	return true, nil
}

// Sweeper deletes orphan blobs past TTL every interval (plus manual
// POST /v1/sweep/run). S3 keys first, then DB rows — crash-safe order.
type Sweeper struct {
	cfg   config.Config
	store *store.Store
	s3    *s3.Client
}

// NewSweeper builds the GC loop.
func NewSweeper(cfg config.Config, st *store.Store, s3c *s3.Client) *Sweeper {
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

// Once runs one bounded sweep pass.
func (s *Sweeper) Once(ctx context.Context) error {
	orphans, err := s.store.SweepOrphans(ctx, 7*24*time.Hour, s.cfg.SweepLimit)
	if err != nil {
		return err
	}
	done := [][32]byte{}
	var bytes int64
	for _, o := range orphans {
		if err := s.s3.Delete(ctx, hash.S3Key(o.Voice, o.Rate, hash.Hex(o.Hash))); err != nil {
			continue
		}
		done = append(done, o.Hash)
		bytes += int64(o.Bytes)
	}
	if len(done) == 0 {
		return nil
	}
	n, err := s.store.DeleteBlobs(ctx, done)
	if err != nil {
		return err
	}
	log.Printf("audio sweep: deleted_blobs=%d deleted_bytes=%d", n, bytes)
	return nil
}
