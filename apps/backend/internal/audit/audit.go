package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogEntry represents an audit log record to be inserted into audit_logs table.
type LogEntry struct {
	ActorUserID  *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *string
	Metadata     map[string]any
	IP           string
	UserAgent    string
	RequestID    string
}

// AuditLog represents a persisted audit log record with pagination fields.
type AuditLog struct {
	ID           int64          `json:"id"`
	ActorUserID  *uuid.UUID     `json:"actor_user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	Metadata     map[string]any `json:"metadata"`
	IP           *string        `json:"ip,omitempty"`
	UserAgent    *string        `json:"user_agent,omitempty"`
	RequestID    *string        `json:"request_id,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// QueryFilter contains parameters to filter audit logs.
type QueryFilter struct {
	Page         int
	PerPage      int
	Action       string
	ResourceType string
	ActorUserID  *uuid.UUID
	DateFrom     *time.Time
	DateTo       *time.Time
}

// Recorder handles recording and querying audit logs in PostgreSQL.
type Recorder struct {
	db *pgxpool.Pool
}

// NewRecorder returns a new audit log recorder.
func NewRecorder(db *pgxpool.Pool) *Recorder {
	return &Recorder{db: db}
}

// Record inserts a new audit log entry into the database.
func (r *Recorder) Record(ctx context.Context, entry LogEntry) error {
	const q = `
		INSERT INTO audit_logs (
			actor_user_id, action, resource_type, resource_id, metadata, ip, user_agent, request_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, NULLIF($6, '')::inet, $7, $8, NOW()
		)
	`
	metaJSON, err := json.Marshal(entry.Metadata)
	if err != nil || entry.Metadata == nil {
		metaJSON = []byte("{}")
	}

	_, err = r.db.Exec(ctx, q,
		entry.ActorUserID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		metaJSON,
		entry.IP,
		entry.UserAgent,
		entry.RequestID,
	)
	if err != nil {
		return fmt.Errorf("inserting audit log: %w", err)
	}

	return nil
}

// RecordFromRequest is a convenience method that infers actor, request ID, IP, and user agent from the HTTP request.
func (r *Recorder) RecordFromRequest(req *http.Request, action, resourceType string, resourceID *string, metadata map[string]any) error {
	var actorID *uuid.UUID
	if p, ok := rbac.GetPrincipal(req.Context()); ok && p != nil {
		actorID = &p.UserID
	}

	reqID := middleware.RequestIDFromContext(req.Context())
	ip := req.RemoteAddr
	if forwarded := req.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	return r.Record(req.Context(), LogEntry{
		ActorUserID:  actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IP:           ip,
		UserAgent:    req.UserAgent(),
		RequestID:    reqID,
	})
}

// Query fetches a paginated list of audit logs according to the provided filter.
func (r *Recorder) Query(ctx context.Context, f QueryFilter) ([]AuditLog, httpx.Meta, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}

	var (
		whereClauses []string
		args         []any
		argIdx       = 1
	)

	if f.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, f.Action)
		argIdx++
	}
	if f.ResourceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, f.ResourceType)
		argIdx++
	}
	if f.ActorUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("actor_user_id = $%d", argIdx))
		args = append(args, *f.ActorUserID)
		argIdx++
	}
	if f.DateFrom != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *f.DateFrom)
		argIdx++
	}
	if f.DateTo != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *f.DateTo)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereSQL)
	var total int
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, httpx.Meta{}, fmt.Errorf("counting audit logs: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + f.PerPage - 1) / f.PerPage
	}

	offset := (f.Page - 1) * f.PerPage
	querySQL := fmt.Sprintf(`
		SELECT id, actor_user_id, action, resource_type, resource_id, metadata, ip::text, user_agent, request_id, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, f.PerPage, offset)

	rows, err := r.db.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, httpx.Meta{}, fmt.Errorf("querying audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var (
			l       AuditLog
			metaRaw []byte
		)
		if err := rows.Scan(
			&l.ID,
			&l.ActorUserID,
			&l.Action,
			&l.ResourceType,
			&l.ResourceID,
			&metaRaw,
			&l.IP,
			&l.UserAgent,
			&l.RequestID,
			&l.CreatedAt,
		); err != nil {
			return nil, httpx.Meta{}, fmt.Errorf("scanning audit log row: %w", err)
		}

		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &l.Metadata)
		}
		if l.Metadata == nil {
			l.Metadata = make(map[string]any)
		}

		logs = append(logs, l)
	}

	if logs == nil {
		logs = []AuditLog{}
	}

	meta := httpx.Meta{
		Page:       f.Page,
		PerPage:    f.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}

	return logs, meta, nil
}
