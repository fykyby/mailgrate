package app

import (
	"app/jobs"
	"app/worker"
)

func RegisterJobs() {
	worker.RegisterJob(jobs.ExampleType, func() worker.JobHandler {
		return jobs.NewExample()
	})
}
