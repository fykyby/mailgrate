package jobs

import (
	"app/worker"
	"context"
	"encoding/json"
	"log/slog"
)

var HelloWorldType = "example_job"

type HelloWorldPayload struct {
	Message string `json:"message"`
}

func HelloWorld() *worker.WorkerJob {
	return &worker.WorkerJob{
		Type: HelloWorldType,
		Run: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
			var data HelloWorldPayload
			json.Unmarshal(payload, &data)

			select {
			case <-ctx.Done():
				json, _ := json.Marshal(data)
				return json, ctx.Err()
			default:
			}

			slog.Info(data.Message)

			json, _ := json.Marshal(data)
			return json, nil
		},
	}
}
