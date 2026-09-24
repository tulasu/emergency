package calls

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CallStore persists calls; the service depends on the interface so tests
// can fake the repo (ARI/dialog stay fake HTTP servers).
type CallStore interface {
	ActiveForAttempt(ctx context.Context, attemptID uuid.UUID) (Call, bool, error)
	Create(ctx context.Context, c Call) error
	FindByID(ctx context.Context, id uuid.UUID) (Call, error)
	ListByAttempt(ctx context.Context, attemptID uuid.UUID) ([]Call, error)
	ListExpired(ctx context.Context, now time.Time) ([]Call, error)
	SetChannelID(ctx context.Context, id uuid.UUID, channelID string) error
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	SaveTurns(ctx context.Context, callID uuid.UUID, turns []Turn) error
}

// Service is the call owner: generates call_id, validates the snapshot on
// dialog BEFORE ringing, then originates via ARI. Ticketgen never dials.
type Service struct {
	Calls     CallStore
	ARI       *ARI
	DialogURL string
	// Loaders injected from tickets/dialog to avoid import cycles.
	LoadAttempt func(ctx context.Context, attemptID, actorID uuid.UUID) (ticketID, userID uuid.UUID, deadline *time.Time, status string, err error)
	LoadTicket  func(ctx context.Context, ticketID uuid.UUID) (scenarioID, scenarioJSON, bankDigest string, err error)
	// MarkAttemptTimedOut closes the attempt row on deadline expiry.
	MarkAttemptTimedOut func(ctx context.Context, attemptID uuid.UUID) error
	// ResolveEndpoint maps a user to their SIP endpoint (user_sip_endpoints).
	// Used when RequestCall.To is empty so `to` is never trusted blindly (spec H).
	ResolveEndpoint func(ctx context.Context, userID uuid.UUID) (string, error)
}

func (s *Service) dialogBase() string {
	if s.DialogURL != "" {
		return s.DialogURL
	}
	return os.Getenv("DIALOG_URL")
}

type RequestCallInput struct {
	AttemptID uuid.UUID
	ActorID   uuid.UUID
	To        string // student SIP endpoint (from user_sip_endpoints)
}

// RequestCall generates call_id = session_id = AudioSocket UUID (AD-6).
// Drifted snapshots are rejected by dialog BEFORE originate: the student
// phone never rings on a bad scenario (spec G). Repeat after
// answered/completed → 409 (BlocksRecall). no_answer/failed leave the
// attempt in_progress so recall is allowed.
func (s *Service) RequestCall(ctx context.Context, in RequestCallInput) (Call, error) {
	if in.ActorID == uuid.Nil {
		return Call{}, ErrInvalidInput // unresolved actor never dials (spec H)
	}
	ticketID, userID, deadline, attemptStatus, err := s.LoadAttempt(ctx, in.AttemptID, in.ActorID)
	if err != nil {
		return Call{}, err
	}
	if attemptStatus != "in_progress" {
		return Call{}, ErrConflict
	}
	if deadline != nil && time.Now().UTC().After(*deadline) {
		return Call{}, ErrGone
	}
	if existing, found, err := s.Calls.ActiveForAttempt(ctx, in.AttemptID); err != nil {
		return Call{}, err
	} else if found && existing.BlocksRecall() {
		return Call{}, ErrConflict
	}
	to := strings.TrimSpace(in.To)
	if to == "" && s.ResolveEndpoint != nil {
		to, err = s.ResolveEndpoint(ctx, userID)
		if err != nil {
			return Call{}, err
		}
		to = strings.TrimSpace(to)
	}
	if to == "" {
		return Call{}, ErrInvalidInput // empty To never dials a blank PJSIP endpoint (spec I)
	}
	if strings.ContainsAny(to, ",; \t\r\n") {
		return Call{}, ErrInvalidInput // endpoint charset: no ARI/dialplan injection via PJSIP/ concat
	}
	scenarioID, scenarioJSON, bankDigest, err := s.LoadTicket(ctx, ticketID)
	if err != nil {
		return Call{}, err
	}
	if strings.TrimSpace(bankDigest) == "" {
		return Call{}, ErrBadSnapshot // empty digest would skip the digest gate: fail closed
	}
	call := Call{
		ID:         uuid.New(),
		AttemptID:  in.AttemptID,
		TicketID:   ticketID,
		UserID:     userID,
		ScenarioID: scenarioID,
		BankDigest: bankDigest,
		Status:     StatusOriginating,
	}
	if err := s.Calls.Create(ctx, call); err != nil {
		return Call{}, err
	}
	fail := func(err error) (Call, error) {
		_ = s.Calls.SetStatus(ctx, call.ID, StatusFailed)
		return Call{}, err
	}
	// Dialog first: 400 on drift, nothing rings.
	if err := s.openDialog(ctx, call, scenarioJSON); err != nil {
		return fail(err)
	}
	channelID, err := s.ARI.Originate(to, call.ID.String())
	if err != nil {
		_ = s.closeDialogSession(ctx, call.ID.String())
		return fail(fmt.Errorf("originate: %w", err))
	}
	if channelID != "" {
		_ = s.Calls.SetChannelID(ctx, call.ID, channelID)
	}
	if err := s.Calls.SetStatus(ctx, call.ID, StatusRinging); err != nil {
		return Call{}, err
	}
	call.Status = StatusRinging
	return call, nil
}

func (s *Service) openDialog(ctx context.Context, call Call, scenarioJSON string) error {
	base := strings.TrimSpace(s.dialogBase())
	if base == "" {
		return fmt.Errorf("dialog worker not configured (DIALOG_URL)")
	}
	if !json.Valid([]byte(scenarioJSON)) || strings.TrimSpace(scenarioJSON) == "" {
		return ErrBadSnapshot
	}
	body, _ := json.Marshal(map[string]any{
		"session_id":  call.ID.String(),
		"scenario":    json.RawMessage(scenarioJSON),
		"bank_digest": call.BankDigest,
	})
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // never hang the ring on dialog (spec I)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimSuffix(base, "/")+"/sessions/open", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 {
		return ErrBadSnapshot
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dialog open: %s", resp.Status)
	}
	return nil
}

// closeDialogSession cleans the dialog session when originate fails after open.
func (s *Service) closeDialogSession(ctx context.Context, sessionID string) error {
	base := strings.TrimSpace(s.dialogBase())
	if base == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"session_id": sessionID})
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimSuffix(base, "/")+"/sessions/close", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// OnEvent applies ARI outcomes. Deadline race: ARI hangup + timed_out,
// partial turns already saved via SaveTurns on close.
// Terminal states never revert (spec P).
func (s *Service) OnEvent(ctx context.Context, callID uuid.UUID, event string) (Call, error) {
	c, err := s.Calls.FindByID(ctx, callID)
	if err != nil {
		return Call{}, err
	}
	if IsTerminal(c.Status) {
		return c, ErrConflict
	}
	var status string
	switch event {
	case "answered":
		status = StatusAnswered
	case "completed":
		status = StatusCompleted
	case "no_answer":
		status = StatusNoAnswer
	case "failed":
		status = StatusFailed
	case "timed_out":
		status = StatusTimedOut
		_ = s.hangupLeg(c)
	default:
		return c, fmt.Errorf("unknown event %q: %w", event, ErrInvalidInput)
	}
	if err := s.Calls.SetStatus(ctx, callID, status); err != nil {
		return Call{}, err
	}
	c.Status = status
	return c, nil
}

// Hangup deletes the call leg (DELETE /calls/{id} with hangup).
// Unknown ids are 404, not silent success (spec P).
func (s *Service) Hangup(ctx context.Context, callID uuid.UUID) error {
	c, err := s.Calls.FindByID(ctx, callID)
	if err != nil {
		return err
	}
	if IsTerminal(c.Status) {
		return nil // idempotent: leg already down
	}
	if err := s.hangupLeg(c); err != nil {
		return err // ARI failure surfaces; the leg is NOT marked completed
	}
	return s.Calls.SetStatus(ctx, callID, StatusCompleted)
}

// hangupLeg targets the stored ARI channel id, falling back to call_id
// best-effort (spec F).
func (s *Service) hangupLeg(c Call) error {
	target := c.ChannelID
	if target == "" {
		target = c.ID.String()
	}
	return s.ARI.Hangup(target)
}

// WatchDeadlines hangs up calls whose attempt deadline passed (spec Q):
// expired calls never stay up waiting for an explicit event.
func (s *Service) WatchDeadlines(ctx context.Context, every time.Duration, now func() time.Time) {
	if every <= 0 {
		every = 15 * time.Second
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.expireOnce(ctx, now())
		}
	}
}

func (s *Service) expireOnce(ctx context.Context, now time.Time) {
	expired, err := s.Calls.ListExpired(ctx, now)
	if err != nil {
		return
	}
	for _, c := range expired {
		_ = s.hangupLeg(c)
		_ = s.Calls.SetStatus(ctx, c.ID, StatusTimedOut)
		if s.MarkAttemptTimedOut != nil {
			_ = s.MarkAttemptTimedOut(ctx, c.AttemptID)
		}
	}
}
