package models

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
	JobStatusPending     JobStatus = "pending"
	JobStatusRunning     JobStatus = "running"
	JobStatusInterrupted JobStatus = "interrupted"
	JobStatusCompleted   JobStatus = "completed"
	JobStatusFailed      JobStatus = "failed"
)

type Job struct {
	bun.BaseModel `bun:"table:jobs"`

	ID           int `bun:",pk,autoincrement"`
	UserID       int
	RelatedTable string `bun:",nullzero"`
	RelatedID    int    `bun:",nullzero"`
	Type         JobType
	Status       JobStatus
	Payload      json.RawMessage `bun:"type:jsonb"` // Payload is mutable and represents job progress
	CreatedAt    time.Time       `bun:",nullzero,default:current_timestamp"`
	StartedAt    time.Time       `bun:",nullzero"`
	FinishedAt   time.Time       `bun:",nullzero"`
	Error        string          `bun:",nullzero"`
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

func CreateJobWithRelated(ctx context.Context, userID int, jobType JobType, relatedTable string, relatedID int, payload json.RawMessage) (*Job, error) {
	job := &Job{
		UserID:       userID,
		Type:         jobType,
		Status:       JobStatusPending,
		Payload:      payload,
		RelatedTable: relatedTable,
		RelatedID:    relatedID,
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

func CreateJobs(ctx context.Context, userID int, jobType JobType, payload []json.RawMessage) ([]*Job, error) {
	jobs := make([]*Job, 0, len(payload))

	for _, p := range payload {
		job := &Job{
			UserID:  userID,
			Type:    jobType,
			Status:  JobStatusPending,
			Payload: p,
		}

		jobs = append(jobs, job)
	}

	_, err := db.Bun.
		NewInsert().
		Model(jobs).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func FindJobByID(ctx context.Context, id int) (*Job, error) {
	job := new(Job)

	err := db.Bun.
		NewSelect().
		Model(job).
		Where("id = ?", id).
		Scan(ctx)

	return job, err
}

func FindJobsByIDs(ctx context.Context, ids []int) ([]*Job, error) {
	jobs := make([]*Job, 0, len(ids))

	err := db.Bun.
		NewSelect().
		Model(&jobs).
		Where("id IN (?)", bun.In(ids)).
		Scan(ctx)

	return jobs, err
}

func FindJobsByRelated(ctx context.Context, relatedTable string, relatedID int) ([]*Job, error) {
	jobs := make([]*Job, 0)

	err := db.Bun.
		NewSelect().
		Model(&jobs).
		Where("related_table = ?", relatedTable).
		Where("related_id = ?", relatedID).
		Scan(ctx)

	return jobs, err
}

func FindPendingJob(ctx context.Context) (*Job, error) {
	job := new(Job)

	err := db.Bun.
		NewSelect().
		Model(job).
		Where("status = ?", JobStatusPending).
		OrderBy("created_at", bun.OrderAsc).
		Limit(1).
		Scan(ctx)

	return job, err
}

func UpdateJob(ctx context.Context, job *Job) error {
	_, err := db.Bun.
		NewUpdate().
		Model(job).
		WherePK().
		OmitZero().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func UpdateJobs(ctx context.Context, jobs []*Job) error {
	_, err := db.Bun.
		NewUpdate().
		Model(&jobs).
		WherePK().
		OmitZero().
		Bulk().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
