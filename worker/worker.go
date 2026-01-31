package worker

import (
	"app/data"
	"app/errorsx"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"
)

type JobHandler interface {
	Run(ctx context.Context) error
}

var jobRegistry = make(map[data.JobType]JobHandler)

func RegisterJob(jobType data.JobType, job JobHandler) {
	jobRegistry[jobType] = job
}

func StartWorker(ctx context.Context, notifyChan <-chan pgdriver.Notification) {
	slog.Info("worker: started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker: stopped accepting jobs")
			return
		case notification := <-notifyChan:
			slog.Debug("worker: received notification", "notification", notification)
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
		err := fmt.Errorf("worker: job type not found: %s", job.Type)
		slog.Error("worker: job type not found", "error", err.Error())
		return err
	}

	err := json.Unmarshal(job.Payload, handler)
	if err != nil {
		slog.Error("worker: failed to unmarshal job payload", "error", err.Error())
		return err
	}

	// Mark job as running
	job.Status = data.JobStatusRunning
	job.StartedAt = time.Now()
	err = updateJob(ctx, job, handler)
	if err != nil {
		return err
	}

	// Run the job
	err = handler.Run(ctx)
	if err != nil && errors.Is(err, context.Canceled) {
		job.Status = data.JobStatusPaused
		err = updateJob(ctx, job, handler)
		if err != nil {
			return err
		}
	} else if err != nil {
		job.Status = data.JobStatusFailed
		job.FinishedAt = time.Now()
		err = updateJob(ctx, job, handler)
		if err != nil {
			return err
		}
	}

	// Mark job as completed
	job.Status = data.JobStatusCompleted
	job.FinishedAt = time.Now()
	err = updateJob(ctx, job, handler)
	if err != nil {
		return err
	}

	return nil
}

func updateJob(ctx context.Context, job *data.Job, handler JobHandler) error {
	payload, err := json.Marshal(handler)
	if err != nil {
		slog.Error("worker: failed to marshal job handler", "error", err.Error())
		return err
	}

	job.Payload = payload

	_, err = data.UpdateJob(ctx, job)
	if err != nil {
		slog.Error("worker: failed to update job", "error", err.Error())
		return err
	}

	return nil
}
