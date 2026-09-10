package reindex

import (
	"time"

	"github.com/google/uuid"
)

type CreateJobRequest struct {
	ToModelVersion string `json:"to_model_version"`
	DryRun         bool   `json:"dry_run"`
}

type CreateJobResponse struct {
	ID                    *uuid.UUID `json:"id,omitempty"`
	FromModelVersion      string     `json:"from_model_version"`
	ToModelVersion        string     `json:"to_model_version"`
	Status                string     `json:"status"`
	TotalCount            int        `json:"total_count"`
	AffectedEmployeeCount int        `json:"affected_employee_count"`
	DryRun                bool       `json:"dry_run,omitempty"`
}

type IncompleteEmployee struct {
	EmployeeID     uuid.UUID `json:"employee_id"`
	EmployeeNumber string    `json:"employee_number"`
	Succeeded      int       `json:"succeeded"`
	Required       int       `json:"required"`
}

type JobDetailResponse struct {
	ID                       uuid.UUID            `json:"id"`
	FromModelVersion         string               `json:"from_model_version"`
	ToModelVersion           string               `json:"to_model_version"`
	Status                   string               `json:"status"`
	TotalCount               int                  `json:"total_count"`
	ProcessedCount           int                  `json:"processed_count"`
	SucceededCount           int                  `json:"succeeded_count"`
	FailedCount              int                  `json:"failed_count"`
	EmployeesReadyCount      int                  `json:"employees_ready_count"`
	EmployeesIncompleteCount int                  `json:"employees_incomplete_count"`
	Error                    *string              `json:"error,omitempty"`
	StartedAt                *time.Time           `json:"started_at"`
	FinishedAt               *time.Time           `json:"finished_at"`
	CreatedAt                time.Time            `json:"created_at"`
	IncompleteEmployees      []IncompleteEmployee `json:"incomplete_employees"`
}
