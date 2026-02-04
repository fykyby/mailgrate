package models

import (
	"app/db"
	"app/helpers"
	"context"

	"github.com/uptrace/bun"
)

type EmailAccount struct {
	bun.BaseModel `bun:"table:email_accounts"`

	Id              int `bun:",pk,autoincrement"`
	SyncListId      int
	SrcUser         string
	SrcPasswordHash string
	DstUser         string
	DstPasswordHash string
}

type EmailAccountsPaginated struct {
	EmailAccounts []*EmailAccount
	Pagination    helpers.Pagination
}

func CreateEmailAccount(ctx context.Context, syncListId int, srcUser string, srcPasswordHash string, dstUser string, dstPasswordHash string) (*EmailAccount, error) {
	emailAccount := &EmailAccount{
		SyncListId:      syncListId,
		SrcUser:         srcUser,
		SrcPasswordHash: srcPasswordHash,
		DstUser:         dstUser,
		DstPasswordHash: dstPasswordHash,
	}

	_, err := db.Bun.
		NewInsert().
		Model(emailAccount).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return emailAccount, nil
}

func FindEmailAccountByID(ctx context.Context, id int) (*EmailAccount, error) {
	emailAccount := new(EmailAccount)

	err := db.Bun.
		NewSelect().
		Model(emailAccount).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return emailAccount, nil
}

func FindEmailAccountsBySyncListIDPaginated(ctx context.Context, syncListId int, page int) (*EmailAccountsPaginated, error) {
	var emailAccounts []*EmailAccount

	err := db.Bun.
		NewSelect().
		Model(&emailAccounts).
		Where("sync_list_id = ?", syncListId).
		Limit(helpers.PaginationLimit).
		Offset((page-1)*helpers.PaginationLimit).
		OrderBy("src_user", bun.OrderAsc).
		OrderBy("dst_user", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	total, err := db.Bun.
		NewSelect().
		Model(&EmailAccount{}).
		Where("sync_list_id = ?", syncListId).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	emailAccountsPaginated := &EmailAccountsPaginated{
		EmailAccounts: emailAccounts,
		Pagination:    helpers.NewPagination(page, total),
	}

	return emailAccountsPaginated, nil
}

func FindEmailAccountsBySyncListID(ctx context.Context, syncListId int) ([]*EmailAccount, error) {
	var emailAccounts []*EmailAccount

	err := db.Bun.
		NewSelect().
		Model(&emailAccounts).
		Where("sync_list_id = ?", syncListId).
		Limit(helpers.PaginationLimit).
		OrderBy("src_user", bun.OrderAsc).
		OrderBy("dst_user", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return emailAccounts, nil
}

func DeleteEmailAccount(ctx context.Context, id int) error {
	emailAccount := &EmailAccount{Id: id}

	_, err := db.Bun.
		NewDelete().
		Model(emailAccount).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	_ = DeleteJobsByRelated(ctx, "email_accounts", id)

	return nil
}
