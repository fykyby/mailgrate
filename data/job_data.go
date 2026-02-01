package data

import (
	"app/db"
	"context"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type JobType string

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusExited    JobStatus = "exited"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	bun.BaseModel `bun:"table:jobs"`

	ID         int `bun:",pk,autoincrement"`
	UserID     int
	Type       JobType
	Status     JobStatus
	Payload    json.RawMessage `bun:"type:jsonb"`
	CreatedAt  time.Time       `bun:",nullzero,default:current_timestamp"`
	StartedAt  time.Time       `bun:",nullzero"`
	FinishedAt time.Time       `bun:",nullzero"`
}

func CreateJob(ctx context.Context, userID int, jobType JobType, payload json.RawMessage) (*Job, error) {
	job := &Job{
		UserID:  userID,
		Type:    jobType,
		Status:  JobStatusPending,
		Payload: payload,
	}

	_, err := db.Bun.
		NewInsert().
		Model(job).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func FindPendingJob(ctx context.Context) (*Job, error) {
	job := new(Job)

	err := db.Bun.
		NewSelect().
		Model(job).
		Where("status = ?", JobStatusExited).
		WhereOr("status = ?", JobStatusPending).
		OrderBy("created_at", bun.OrderAsc).
		Limit(1).
		Scan(ctx)

	return job, err
}

func UpdateJob(ctx context.Context, job *Job) (*Job, error) {
	_, err := db.Bun.
		NewUpdate().
		Model(job).
		WherePK().
		OmitZero().
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return job, nil
}
