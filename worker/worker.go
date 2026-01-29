package worker

import (
	"app/httpx"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

type Job interface {
	ID() string
	Run(ctx context.Context)
}

var workerCount = 0
var jobs chan Job

func InitJobQueue() {
	jobQueueSizeStr := os.Getenv("JOB_QUEUE_SIZE")
	jobQueueSize, err := strconv.Atoi(jobQueueSizeStr)
	if err != nil || jobQueueSize < 0 {
		slog.Error("invalid job queue size: " + err.Error())
		return
	} else if jobQueueSize == 0 {
		return
	}

	jobs = make(chan Job, jobQueueSize)
}

func StartWorker(ctx context.Context) {
	workerCount++
	slog.Info("worker: started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker: stopped")
			workerCount--
			return
		case job := <-jobs:
			slog.Info(fmt.Sprintf("worker: starting job %s", job.ID()))
			job.Run(ctx)
			slog.Info(fmt.Sprintf("worker: completed job %s", job.ID()))
		}
	}
}

func AddJob(job Job) error {
	if workerCount == 0 || cap(jobs) <= 0 {
		return errors.New(httpx.MsgErrWorkersUnavailable)
	}

	select {
	case jobs <- job:
		return nil
	default:
		return errors.New(httpx.MsgErrJobQueueFull)
	}
}
