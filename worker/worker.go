package worker

import (
	"app/config"
	"app/errorsx"
	"app/models"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"
)

type JobHandler interface {
	Run(ctx context.Context) error
	OnStop(ctx context.Context) error
}

type runningJob struct {
	Handler JobHandler
	Cancel  context.CancelFunc
}

type jobFactory func(ctx context.Context, payload *json.RawMessage) (JobHandler, error)

var runningJobs = map[int]*runningJob{}
var runningJobsMu sync.Mutex

// Immutable after app.RegisterJobs()
var jobHandlerRegistry = make(map[models.JobType]jobFactory)

func RegisterJob(jobType models.JobType, factory jobFactory) {
	jobHandlerRegistry[jobType] = factory
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

func runJob(workerCtx context.Context, job *models.Job) {
	var err error = nil
	defer func() {
		r := recover()
		if r != nil {
			stack := debug.Stack()
			slog.Error("worker: job panicked", "job", job.Id, "panic", r, "stack", string(stack))
			err = errors.New("job panicked")
		}

		switch {
		case errors.Is(err, context.Canceled):
			slog.Info("worker: job interrupted", "job", job.Id, "error", err.Error())
			job.Error = nil
			job.Status = models.JobStatusInterrupted
		case err != nil:
			slog.Error("worker: job failed", "job", job.Id, "error", err.Error())
			errStr := err.Error()
			job.Error = &errStr
			job.Status = models.JobStatusFailed
		default:
			slog.Info("worker: job completed", "job", job.Id)
			job.Error = nil
			job.Status = models.JobStatusCompleted
		}

		now := time.Now()
		job.FinishedAt = &now

		err = models.UpdateJob(workerCtx, job)
		if err != nil {
			slog.Error("worker: failed to update stopping job", "job", job.Id, "error", err.Error())
			return
		}

		err = GetRunningJob(job.Id).Handler.OnStop(workerCtx)
		if err != nil {
			slog.Error("worker: failed to run OnStop", "job", job.Id, "error", err.Error())
			return
		}

		runningJobsMu.Lock()
		delete(runningJobs, job.Id)
		runningJobsMu.Unlock()
	}()

	slog.Info("worker: job started", "job", job.Id)

	now := time.Now()
	job.StartedAt = &now
	job.Status = models.JobStatusRunning
	err = models.UpdateJob(workerCtx, job)
	if err != nil {
		slog.Error("worker: failed to update starting job", "job", job.Id, "error", err.Error())
		return
	}

	jobCtx, cancelJob := context.WithTimeout(workerCtx, time.Duration(config.Config.JobTimeoutMinutes)*time.Minute)
	defer cancelJob()

	factory, ok := jobHandlerRegistry[job.Type]
	if !ok {
		slog.Error("worker: unknown job type", "job", job.Id)
		err = errors.New("unknown job type")
		return
	}
	handler, err := factory(jobCtx, job.Payload)

	runningJobsMu.Lock()
	runningJobs[job.Id] = &runningJob{
		Handler: handler,
		Cancel:  cancelJob,
	}
	runningJobsMu.Unlock()

	err = handler.Run(jobCtx)
}
