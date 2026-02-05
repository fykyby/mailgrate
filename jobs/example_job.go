package jobs

import (
	"app/models"
	"context"
	"fmt"
	"log/slog"
	"time"
)

var ExampleType models.JobType = "example"

type Example struct{}

func NewExample() *Example {
	return &Example{}
}

func (j *Example) Run(ctx context.Context) error {
	for i := range 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slog.Info(fmt.Sprintf("count: %d", i))

		time.Sleep(time.Second)

	}

	return nil
}

func (j *Example) OnStop(ctx context.Context, err error) error {
	return nil
}
