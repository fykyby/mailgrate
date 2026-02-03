package models

import (
	"app/db"
	"app/helpers"
	"context"

	"github.com/uptrace/bun"
)

type EmailAccount struct {
	bun.BaseModel `bun:"table:email_accounts"`

	ID         int `bun:",pk,autoincrement"`
	SyncListID int
	Login      string
	Password   string
}

type EmailAccountsPaginated struct {
	EmailAccounts []*EmailAccount
	Pagination    helpers.Pagination
}

type EmailAccountsJobs struct {
	bun.BaseModel `bun:"table:email_accounts_jobs"`

	ID             int `bun:",pk,autoincrement"`
	EmailAccountID int
	JobID          int
}

func CreateEmailAccount(ctx context.Context, syncListID int, login string, password string) (*EmailAccount, error) {
	emailAccount := &EmailAccount{
		SyncListID: syncListID,
		Login:      login,
		Password:   password,
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

func FindEmailAccountsBySyncListIDPaginated(ctx context.Context, syncListID int, page int) (*EmailAccountsPaginated, error) {
	var emailAccounts []*EmailAccount

	err := db.Bun.
		NewSelect().
		Model(&emailAccounts).
		Where("sync_list_id = ?", syncListID).
		Limit(helpers.PaginationLimit).
		Offset((page-1)*helpers.PaginationLimit).
		OrderBy("login", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	total, err := db.Bun.
		NewSelect().
		Model(&EmailAccount{}).
		Where("sync_list_id = ?", syncListID).
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

func DeleteEmailAccount(ctx context.Context, id int) error {
	emailAccount := &EmailAccount{ID: id}

	_, err := db.Bun.
		NewDelete().
		Model(emailAccount).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
