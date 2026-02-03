package worker

import (
	"app/config"
	"app/errorsx"
	"app/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"
)

type JobHandler interface {
	Run(ctx context.Context) error
}

type runningJob struct {
	Handler JobHandler
	Cancel  context.CancelFunc
}

var runningJobs = map[int]*runningJob{}
var runningJobsMu sync.Mutex

// Immutable after app.RegisterJobs()
var jobRegistry = make(map[models.JobType]func() JobHandler)

func RegisterJob(jobType models.JobType, factory func() JobHandler) {
	jobRegistry[jobType] = factory
}

func GetRunningJob(id int) *runningJob {
	runningJobsMu.Lock()
	defer runningJobsMu.Unlock()

	v, ok := runningJobs[id]
	if !ok {
		return nil
	}
	return v
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
			job, err := models.FindPendingJob(ctx)
			if err != nil {
				if !errorsx.IsNotFoundError(err) {
					slog.Error("worker: failed to find job", "error", err)
				}
				continue
			}

			runJob(ctx, job)
		}
	}
}

func runJob(ctx context.Context, job *models.Job) {
	slog.Info("worker: job started", "job", job.ID)

	jobCtx, cancelJob := context.WithTimeout(ctx, time.Duration(config.Config.JobTimeoutMinutes)*time.Minute)
	defer cancelJob()

	factory, ok := jobRegistry[job.Type]
	if !ok {
		slog.Error("worker: job type not found", "job", job.ID)

		job.Status = models.JobStatusFailed
		job.Error = "unknown job type"
		job.FinishedAt = time.Now()
		_ = updateJob(ctx, job, nil)
		return
	}
	handler := factory()

	runningJobsMu.Lock()
	runningJobs[job.ID] = &runningJob{
		Handler: handler,
		Cancel:  cancelJob,
	}
	runningJobsMu.Unlock()
	defer func() {
		runningJobsMu.Lock()
		delete(runningJobs, job.ID)
		runningJobsMu.Unlock()
	}()

	defer func() {
		r := recover()
		if r != nil {
			stack := debug.Stack()

			slog.Error("worker: job panicked", "job", job.ID, "panic", r, "stack", string(stack))

			job.Status = models.JobStatusFailed
			job.Error = fmt.Sprintf("%v", r)
			job.FinishedAt = time.Now()
			_ = updateJob(ctx, job, handler)
		}
	}()

	// Payload is mutable and represents job progress
	err := json.Unmarshal(job.Payload, handler)
	if err != nil {
		slog.Error("worker: failed to unmarshal job payload", "job", job.ID, "error", err.Error())
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
		job.FinishedAt = time.Now()
		_ = updateJob(ctx, job, nil)
		return
	}

	// Mark job as running
	job.Status = models.JobStatusRunning
	job.StartedAt = time.Now()
	err = updateJob(ctx, job, handler)
	if err != nil {
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
		job.FinishedAt = time.Now()
		updateJob(ctx, job, nil)
		return
	}

	// Run the job
	err = handler.Run(jobCtx)

	switch {
	case errors.Is(err, context.Canceled):
		slog.Info("worker: job interrupted", "job", job.ID, "error", err.Error())
		job.Status = models.JobStatusInterrupted
	case err != nil:
		slog.Error("worker: job failed", "job", job.ID, "error", err.Error())
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
	default:
		slog.Info("worker: job completed", "job", job.ID)
		job.Status = models.JobStatusCompleted
	}

	job.FinishedAt = time.Now()
	err = updateJob(ctx, job, handler)
	if err != nil {
		slog.Error("worker: failed to update job", "job", job.ID, "error", err.Error())
		return
	}
}

func updateJob(ctx context.Context, job *models.Job, handler JobHandler) error {
	if handler != nil {
		payload, err := json.Marshal(handler)
		if err != nil {
			slog.Error("worker: failed to marshal job handler", "job", job.ID, "error", err.Error())
			job.Payload = []byte(`{"error":"failed to marshal handler state"}`)
		} else {
			job.Payload = payload
		}
	}

	err := models.UpdateJob(ctx, job)
	if err != nil {
		slog.Error("worker: failed to update job", "job", job.ID, "error", err.Error())
		return err
	}

	return nil
}
