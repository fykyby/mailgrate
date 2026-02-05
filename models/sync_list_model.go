package models

import (
	"app/db"
	"app/helpers"
	"context"

	"github.com/uptrace/bun"
)

type SyncList struct {
	bun.BaseModel `bun:"table:sync_lists"`

	Id                int `bun:",pk,autoincrement"`
	UserId            int
	Name              string
	SrcHost           string
	SrcPort           int
	DstHost           string
	DstPort           int
	CompareMessageIds bool
	CompareLastUid    bool

	Mailboxes []*Mailbox `bun:"rel:has-many,join:id=sync_list_id"`
}

type SyncListsPaginated struct {
	SyncLists  []*SyncList
	Pagination helpers.Pagination
}

type SyncListWithMailboxesPaginated struct {
	SyncList          *SyncList
	MailboxPagination helpers.Pagination
}

type SyncListStatus struct {
	Id     int
	Status JobStatus
}

type CreateSyncListParams struct {
	UserId            int
	Name              string
	SrcHost           string
	SrcPort           int
	DstHost           string
	DstPort           int
	CompareMessageIds bool
	CompareLastUid    bool
}

func CreateSyncList(ctx context.Context, params CreateSyncListParams) (*SyncList, error) {
	syncList := &SyncList{
		UserId:            params.UserId,
		Name:              params.Name,
		SrcHost:           params.SrcHost,
		SrcPort:           params.SrcPort,
		DstHost:           params.DstHost,
		DstPort:           params.DstPort,
		CompareMessageIds: params.CompareMessageIds,
		CompareLastUid:    params.CompareLastUid,
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

func FindSyncListById(ctx context.Context, id int) (*SyncList, error) {
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

func FindSyncListByIdWithMailboxById(ctx context.Context, id int, mailboxId int) (*SyncList, error) {
	var syncList SyncList

	err := db.Bun.
		NewSelect().
		Model(&syncList).
		Where("id = ?", id).
		Relation("Mailboxes", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("id = ?", mailboxId)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &syncList, nil
}

func FindSyncListByIdWithMailboxes(ctx context.Context, id int) (*SyncList, error) {
	var syncList SyncList

	err := db.Bun.
		NewSelect().
		Model(&syncList).
		Where("id = ?", id).
		Relation("Mailboxes").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &syncList, nil
}

func FindSyncListByIdWithMailboxesPaginated(ctx context.Context, id int, page int) (*SyncListWithMailboxesPaginated, error) {
	var syncList SyncList

	err := db.Bun.
		NewSelect().
		Model(&syncList).
		Where("id = ?", id).
		Relation("Mailboxes", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Limit(helpers.PaginationLimit).
				Offset((page-1)*helpers.PaginationLimit).
				OrderBy("src_user", bun.OrderAsc).
				OrderBy("dst_user", bun.OrderAsc)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	total, err := db.Bun.
		NewSelect().
		Model(&Mailbox{}).
		Where("sync_list_id = ?", id).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	paginatedSyncList := &SyncListWithMailboxesPaginated{
		SyncList:          &syncList,
		MailboxPagination: helpers.NewPagination(page, total),
	}

	return paginatedSyncList, nil
}

func FindSyncListsByUserIdPaginated(ctx context.Context, userId int, page int) (*SyncListsPaginated, error) {
	var syncLists []*SyncList

	err := db.Bun.
		NewSelect().
		Model(&syncLists).
		Where("user_id = ?", userId).
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
		Where("user_id = ?", userId).
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

func DeleteSyncListById(ctx context.Context, id int) error {
	accounts, err := FindMailboxesBySyncListId(ctx, id)
	if err == nil {
		accountIds := make([]int, len(accounts))
		for i, account := range accounts {
			accountIds[i] = account.Id
		}
		_ = DeleteJobsByManyRelated(ctx, "mailboxes", accountIds)
	}

	_, err = db.Bun.
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
		Join("LEFT JOIN mailboxes ea ON sl.id = ea.sync_list_id").
		Join("LEFT JOIN jobs j ON ea.id = j.related_id AND j.related_table = ?", "mailboxes").
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
		Join("LEFT JOIN mailboxes ea ON sl.id = ea.sync_list_id").
		Join("LEFT JOIN jobs j ON ea.id = j.related_id AND j.related_table = ?", "mailboxes").
		Where("sl.id IN (?)", bun.In(ids)).
		GroupExpr("sl.id").
		Scan(ctx, &results)

	return results, err
}
