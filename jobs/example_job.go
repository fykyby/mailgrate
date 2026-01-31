package jobs

import (
	"app/data"
	"context"
	"fmt"
	"log/slog"
)

var ExampleJobType data.JobType = "example_job"

type ExampleJob struct {
	Count int
}

func NewExampleJob() *ExampleJob {
	return &ExampleJob{
		Count: 0,
	}
}

func (j *ExampleJob) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	j.Count++
	slog.Info(fmt.Sprintf("count: %d", j.Count))

	return nil
}
