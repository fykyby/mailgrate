package app

import (
	"app/jobs"
	"app/worker"
)

func RegisterJobs() {
	worker.RegisterJob(jobs.ExampleJobType, jobs.NewExampleJob())
}
