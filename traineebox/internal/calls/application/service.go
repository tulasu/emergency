package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"traineebox/internal/calls/domain/errs"
	"traineebox/internal/calls/domain/models"
	"traineebox/internal/calls/domain/repositories"
	"traineebox/internal/calls/domain/value_objects"

	"github.com/google/uuid"
)

type Dialer interface {
	Originate(to, callID string) (string, error)
	Hangup(target string) error
}

type DialogSessions interface {
	Open(ctx context.Context, call models.Call, scenarioJSON string) error
	Close(ctx context.Context, sessionID string) error
}

type Service struct {
	Calls               repositories.CallRepository
	Dialer              Dialer
	Sessions            DialogSessions
	LoadAttempt         func(ctx context.Context, attemptID, actorID uuid.UUID) (ticketID, userID uuid.UUID, deadline *time.Time, status string, err error)
	LoadTicket          func(ctx context.Context, ticketID uuid.UUID) (scenarioID, scenarioJSON, bankDigest string, err error)
	MarkAttemptTimedOut func(ctx context.Context, attemptID uuid.UUID) error
	ResolveEndpoint     func(ctx context.Context, userID uuid.UUID) (string, error)
}

type RequestCallInput struct {
	AttemptID uuid.UUID
	ActorID   uuid.UUID
	To        string
}

func (s *Service) RequestCall(ctx context.Context, in RequestCallInput) (models.Call, error) {
	if in.ActorID == uuid.Nil {
		return models.Call{}, errs.ErrInvalidInput
	}
	ticketID, userID, deadline, attemptStatus, err := s.LoadAttempt(ctx, in.AttemptID, in.ActorID)
	if err != nil {
		return models.Call{}, fmt.Errorf("load attempt: %w", err)
	}
	if attemptStatus != "in_progress" {
		return models.Call{}, errs.ErrConflict
	}
	if deadline != nil && time.Now().UTC().After(*deadline) {
		return models.Call{}, errs.ErrGone
	}
	if existing, found, err := s.Calls.ActiveForAttempt(ctx, in.AttemptID); err != nil {
		return models.Call{}, fmt.Errorf("active lookup: %w", err)
	} else if found && existing.BlocksRecall() {
		_ = existing
	}
	to := strings.TrimSpace(in.To)
	if to == "" && s.ResolveEndpoint != nil {
		to, err = s.ResolveEndpoint(ctx, userID)
		if err != nil {
			return models.Call{}, fmt.Errorf("resolve endpoint: %w", err)
		}
		to = strings.TrimSpace(to)
	}
	if to == "" {
		return models.Call{}, errs.ErrInvalidInput
	}
	if strings.ContainsAny(to, ",; \t\r\n") {
		return models.Call{}, errs.ErrInvalidInput
	}
	scenarioID, scenarioJSON, bankDigest, err := s.LoadTicket(ctx, ticketID)
	if err != nil {
		return models.Call{}, fmt.Errorf("load ticket: %w", err)
	}
	if strings.TrimSpace(bankDigest) == "" {
		return models.Call{}, errs.ErrBadSnapshot
	}
	call := models.Call{
		ID:         uuid.New(),
		AttemptID:  in.AttemptID,
		TicketID:   ticketID,
		UserID:     userID,
		ScenarioID: scenarioID,
		BankDigest: bankDigest,
		Status:     value_objects.StatusOriginating,
	}
	if err := s.Calls.Create(ctx, call); err != nil {
		return models.Call{}, err
	}
	fail := func(err error) (models.Call, error) {
		_ = s.Calls.SetStatus(ctx, call.ID, value_objects.StatusFailed)
		return models.Call{}, err
	}
	if err := s.Sessions.Open(ctx, call, scenarioJSON); err != nil {
		return fail(err)
	}
	channelID, err := s.Dialer.Originate(to, call.ID.String())
	if err != nil {
		_ = s.Sessions.Close(ctx, call.ID.String())
		return fail(fmt.Errorf("originate: %w", err))
	}
	if channelID != "" {
		_ = s.Calls.SetChannelID(ctx, call.ID, channelID)
	}
	if err := s.Calls.SetStatus(ctx, call.ID, value_objects.StatusRinging); err != nil {
		return models.Call{}, err
	}
	call.Status = value_objects.StatusRinging
	return call, nil
}

func (s *Service) OnEvent(ctx context.Context, callID uuid.UUID, event string) (models.Call, error) {
	c, err := s.Calls.FindByID(ctx, callID)
	if err != nil {
		return models.Call{}, err
	}
	if value_objects.IsTerminal(c.Status) {
		return c, errs.ErrConflict
	}
	var status string
	switch event {
	case "answered":
		status = value_objects.StatusAnswered
	case "completed":
		status = value_objects.StatusCompleted
	case "no_answer":
		status = value_objects.StatusNoAnswer
	case "failed":
		status = value_objects.StatusFailed
	case "timed_out":
		status = value_objects.StatusTimedOut
		_ = s.hangupLeg(c)
	default:
		return c, fmt.Errorf("unknown event %q: %w", event, errs.ErrInvalidInput)
	}
	if err := s.Calls.SetStatus(ctx, callID, status); err != nil {
		return models.Call{}, err
	}
	c.Status = status
	return c, nil
}

func (s *Service) Hangup(ctx context.Context, callID uuid.UUID) error {
	c, err := s.Calls.FindByID(ctx, callID)
	if err != nil {
		return err
	}
	if value_objects.IsTerminal(c.Status) {
		return nil
	}
	if err := s.hangupLeg(c); err != nil {
		return err
	}
	return s.Calls.SetStatus(ctx, callID, value_objects.StatusCompleted)
}

func (s *Service) hangupLeg(c models.Call) error {
	target := c.ChannelID
	if target == "" {
		target = c.ID.String()
	}
	if s.Dialer == nil {
		return nil
	}
	return s.Dialer.Hangup(target)
}

func (s *Service) ListByAttempt(ctx context.Context, attemptID uuid.UUID) ([]models.Call, error) {
	return s.Calls.ListByAttempt(ctx, attemptID)
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (models.Call, error) {
	return s.Calls.FindByID(ctx, id)
}

func (s *Service) SaveTurns(ctx context.Context, callID uuid.UUID, turns []models.Turn) error {
	return s.Calls.SaveTurns(ctx, callID, turns)
}

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
		_ = s.Calls.SetStatus(ctx, c.ID, value_objects.StatusTimedOut)
		if s.MarkAttemptTimedOut != nil {
			_ = s.MarkAttemptTimedOut(ctx, c.AttemptID)
		}
	}
}
