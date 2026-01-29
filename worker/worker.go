package worker

import (
	"app/httpx"
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type Job interface {
	ID() string
	Run(ctx context.Context)
}

var workerCount = 0
var jobs = make(chan Job, 64)

func StartWorker(ctx context.Context) {
	workerCount++
	slog.Info(fmt.Sprintf("worker %d: started", workerCount))
	for {
		select {
		case <-ctx.Done():
			slog.Info(fmt.Sprintf("worker %d: stopped", workerCount))
			workerCount--
			return
		case job := <-jobs:
			slog.Info(fmt.Sprintf("worker %d: starting job %s", workerCount, job.ID()))
			job.Run(ctx)
			slog.Info(fmt.Sprintf("worker %d: job %s completed", workerCount, job.ID()))
		}
	}
}

func AddJob(job Job) error {
	if workerCount == 0 {
		return errors.New(httpx.MsgErrWorkersUnavailable)
	}

	select {
	case jobs <- job:
		return nil
	default:
		return errors.New(httpx.MsgErrJobQueueFull)
	}
}
