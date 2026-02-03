package models

import (
	"app/db"
	"app/helpers"
	"context"

	"github.com/uptrace/bun"
)

type SyncList struct {
	bun.BaseModel `bun:"table:sync_lists"`

	ID              int `bun:",pk,autoincrement"`
	UserID          int
	Name            string
	SourceHost      string
	SourcePort      int
	DestinationHost string
	DestinationPort int
}

type SyncListsPaginated struct {
	SyncLists  []*SyncList
	Pagination helpers.Pagination
}

type SyncListStatus struct {
	ID     int
	Status JobStatus
}

func CreateSyncList(ctx context.Context, userID int, name string, sourceHost string, sourcePort int, destinationHost string, destinationPort int) (*SyncList, error) {
	syncList := &SyncList{
		UserID:          userID,
		Name:            name,
		SourceHost:      sourceHost,
		SourcePort:      sourcePort,
		DestinationHost: destinationHost,
		DestinationPort: destinationPort,
	}

	_, err := db.Bun.
		NewInsert().
		Model(syncList).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return syncList, nil
}

func FindSyncListByID(ctx context.Context, id int) (*SyncList, error) {
	var syncList SyncList

	err := db.Bun.
		NewSelect().
		Model(&syncList).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &syncList, nil
}

func FindSyncListsByUserIDPaginated(ctx context.Context, userID int, page int) (*SyncListsPaginated, error) {
	var syncLists []*SyncList

	err := db.Bun.
		NewSelect().
		Model(&syncLists).
		Where("user_id = ?", userID).
		Limit(helpers.PaginationLimit).
		Offset((page-1)*helpers.PaginationLimit).
		OrderBy("name", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	total, err := db.Bun.
		NewSelect().
		Model(&SyncList{}).
		Where("user_id = ?", userID).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	paginatedSyncLists := &SyncListsPaginated{
		SyncLists:  syncLists,
		Pagination: helpers.NewPagination(page, total),
	}

	return paginatedSyncLists, nil
}

func UpdateSyncList(ctx context.Context, syncList *SyncList) error {
	_, err := db.Bun.
		NewUpdate().
		Model(syncList).
		WherePK().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DeleteSyncListByID(ctx context.Context, id int) error {
	_, err := db.Bun.
		NewDelete().
		Model(&SyncList{}).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func FindSyncListStatus(ctx context.Context, id int) (SyncListStatus, error) {
	var results SyncListStatus

	err := db.Bun.
		NewSelect().
		TableExpr("sync_lists sl").
		ColumnExpr("sl.id").
		ColumnExpr(`
            COALESCE(
                CASE
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = 1 THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = 1 THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = 1 THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = COUNT(*) THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = COUNT(*) THEN ?
                    ELSE ?
                END,
                ?
            ) as status
        `,
			JobStatusRunning, JobStatusRunning,
			JobStatusInterrupted, JobStatusInterrupted,
			JobStatusFailed, JobStatusFailed,
			JobStatusCompleted, JobStatusCompleted,
			JobStatusPending, JobStatusPending,
			JobStatusNone,
			JobStatusNone,
		).
		Join("LEFT JOIN email_accounts ea ON sl.id = ea.sync_list_id").
		Join("LEFT JOIN jobs j ON ea.id = j.related_id AND j.related_table = ?", "email_accounts").
		Where("sl.id = ?", id).
		GroupExpr("sl.id").
		Scan(ctx, &results)

	return results, err
}

func FindSyncListsStatus(ctx context.Context, ids []int) ([]SyncListStatus, error) {
	var results []SyncListStatus

	err := db.Bun.
		NewSelect().
		TableExpr("sync_lists sl").
		ColumnExpr("sl.id").
		ColumnExpr(`
            COALESCE(
                CASE
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = 1 THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = 1 THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = 1 THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = COUNT(*) THEN ?
                    WHEN MAX(CASE WHEN j.status = ? THEN 1 ELSE 0 END) = COUNT(*) THEN ?
                    ELSE ?
                END,
                ?
            ) as status
        `,
			JobStatusRunning, JobStatusRunning,
			JobStatusInterrupted, JobStatusInterrupted,
			JobStatusFailed, JobStatusFailed,
			JobStatusCompleted, JobStatusCompleted,
			JobStatusPending, JobStatusPending,
			JobStatusNone,
			JobStatusNone,
		).
		Join("LEFT JOIN email_accounts ea ON sl.id = ea.sync_list_id").
		Join("LEFT JOIN jobs j ON ea.id = j.related_id AND j.related_table = ?", "email_accounts").
		Where("sl.id IN (?)", bun.In(ids)).
		GroupExpr("sl.id").
		Scan(ctx, &results)

	return results, err
}
