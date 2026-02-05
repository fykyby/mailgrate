package app

import (
	"app/jobs"
	"app/models"
	"app/worker"
	"context"
	"encoding/json"
)

func RegisterJobs() {
	worker.RegisterJob(jobs.MigrateMailboxType, func(ctx context.Context, payload *json.RawMessage) (worker.JobHandler, error) {
		migrateMailboxPayload := new(jobs.MigrateMailboxPayload)

		err := json.Unmarshal(*payload, migrateMailboxPayload)
		if err != nil {
			return nil, err
		}

		list, err := models.FindSyncListById(ctx, migrateMailboxPayload.SyncListId)
		if err != nil {
			return nil, err
		}

		mailbox, err := models.FindMailboxById(ctx, migrateMailboxPayload.MailboxId)
		if err != nil {
			return nil, err
		}

		handler := &jobs.MigrateMailbox{
			SyncList: list,
			Mailbox:  mailbox,
		}

		return handler, nil
	})
}
