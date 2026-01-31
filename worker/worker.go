package worker

import (
	"app/config"
	"app/data"
	"app/errorsx"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type WorkerJob struct {
	Type string
	Run  func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error)
}

var jobRegistry = make(map[string]*WorkerJob)

func RegisterJob(job *WorkerJob) {
	jobRegistry[job.Type] = job
}

func StartWorker(ctx context.Context) {
	slog.Info("worker: started")

	ticker := time.NewTicker(time.Second * time.Duration(config.Config.WorkerPollIntervalSeconds))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker: stopped accepting jobs")
			return
		case <-ticker.C:
			slog.Info("worker: looking for job")
			job, err := data.FindPendingJob(ctx)
			if err != nil {
				if !errorsx.IsNotFoundError(err) {
					slog.Error("worker: failed to find job", "error", err)
				}
				continue
			}

			slog.Info("worker: starting job", "job", job.ID)
			runJob(ctx, job)
			slog.Info("worker: finished job", "job", job.ID)
		}
	}
}

func runJob(ctx context.Context, job *data.Job) error {
	handler, ok := jobRegistry[job.Type]
	if !ok {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}

	// Mark job as running
	if err := updateJobStatus(ctx, job.ID, data.JobStatusRunning, nil); err != nil {
		return err
	}

	// Run the job
	payload, err := handler.Run(ctx, job.Payload)
	if err != nil {
		return handleJobError(ctx, job.ID, err, payload)
	}

	// Mark job as completed
	return updateJobStatus(ctx, job.ID, data.JobStatusCompleted, payload)
}

func updateJobStatus(ctx context.Context, jobID int, status data.JobStatus, payload json.RawMessage) error {
	update := &data.Job{
		ID:      jobID,
		Status:  status,
		Payload: payload,
	}

	switch status {
	case data.JobStatusRunning:
		update.StartedAt = time.Now()
	case data.JobStatusCompleted, data.JobStatusFailed:
		update.FinishedAt = time.Now()
	}

	_, err := data.UpdateJob(ctx, update)
	if err != nil {
		slog.Error("worker: failed to update job status", "error", err.Error())
	}
	return err
}

func handleJobError(ctx context.Context, jobID int, err error, payload json.RawMessage) error {
	if errors.Is(err, context.Canceled) {
		slog.Info("worker: job canceled")
		if e := updateJobStatus(ctx, jobID, data.JobStatusPaused, payload); e != nil {
			return e
		}
		return nil
	}

	slog.Error("worker: failed to run job", "error", err.Error())
	updateJobStatus(ctx, jobID, data.JobStatusFailed, payload)
	return err
}
