package models

import "github.com/google/uuid"

type ReportError struct {
	Field    string `json:"field"`
	Expected any    `json:"expected,omitempty"`
	Actual   any    `json:"actual,omitempty"`
}

type ReportItem struct {
	TicketID uuid.UUID     `json:"ticket_id"`
	Score    int           `json:"score"`
	Errors   []ReportError `json:"errors"`
}

type Report struct {
	OverallScore int          `json:"overall_score"`
	Items        []ReportItem `json:"items"`
}

func EmptyReport() Report {
	return Report{Items: []ReportItem{}}
}
