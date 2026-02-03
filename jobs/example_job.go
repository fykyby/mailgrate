package jobs

import (
	"app/models"
	"context"
	"fmt"
	"log/slog"
	"time"
)

var ExampleType models.JobType = "example"

type Example struct {
	Count int
}

func NewExample() *Example {
	return &Example{
		Count: 0,
	}
}

func (j *Example) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		j.Count++
		slog.Info(fmt.Sprintf("count: %d", j.Count))

		time.Sleep(time.Second)

		if j.Count >= 20 {
			break
		}
	}

	return nil
}
