package app

import (
	"app/jobs"
	"app/worker"
)

func RegisterJobs() {
	worker.RegisterJob(jobs.MigrateAccountType, func() worker.JobHandler {
		return jobs.NewMigrateAccount(jobs.NewMigrateAccountParams{})
	})
}
