package models

import (
	"app/db"
	"app/helpers"
	"context"

	"github.com/uptrace/bun"
)

type Mailbox struct {
	bun.BaseModel `bun:"table:mailboxes"`

	Id                int `bun:",pk,autoincrement"`
	SyncListId        int
	SrcUser           string
	SrcPasswordHash   string
	DstUser           string
	DstPasswordHash   string
	FolderLastUid     map[string]uint32
	FolderUidValidity map[string]uint32

	SyncList *SyncList `bun:"rel:belongs-to,join:sync_list_id=id"`
}

type MailboxesPaginated struct {
	Mailboxes  []*Mailbox
	Pagination helpers.Pagination
}

func CreateMailbox(ctx context.Context, syncListId int, srcUser string, srcPasswordHash string, dstUser string, dstPasswordHash string) (*Mailbox, error) {
	Mailbox := &Mailbox{
		SyncListId:        syncListId,
		SrcUser:           srcUser,
		SrcPasswordHash:   srcPasswordHash,
		DstUser:           dstUser,
		DstPasswordHash:   dstPasswordHash,
		FolderLastUid:     make(map[string]uint32),
		FolderUidValidity: make(map[string]uint32),
	}

	_, err := db.Bun.
		NewInsert().
		Model(Mailbox).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return Mailbox, nil
}

func FindMailboxById(ctx context.Context, id int) (*Mailbox, error) {
	Mailbox := new(Mailbox)

	err := db.Bun.
		NewSelect().
		Model(Mailbox).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return Mailbox, nil
}

func FindMailboxesBySyncListIdPaginated(ctx context.Context, syncListId int, page int) (*MailboxesPaginated, error) {
	var Mailboxes []*Mailbox

	err := db.Bun.
		NewSelect().
		Model(&Mailboxes).
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
		Model((*Mailbox)(nil)).
		Where("sync_list_id = ?", syncListId).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	MailboxesPaginated := &MailboxesPaginated{
		Mailboxes:  Mailboxes,
		Pagination: helpers.NewPagination(page, total),
	}

	return MailboxesPaginated, nil
}

func FindMailboxesBySyncListId(ctx context.Context, syncListId int) ([]*Mailbox, error) {
	var Mailboxes []*Mailbox

	err := db.Bun.
		NewSelect().
		Model(&Mailboxes).
		Where("sync_list_id = ?", syncListId).
		Limit(helpers.PaginationLimit).
		OrderBy("src_user", bun.OrderAsc).
		OrderBy("dst_user", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return Mailboxes, nil
}

func UpdateMailbox(ctx context.Context, Mailbox *Mailbox) error {
	_, err := db.Bun.
		NewUpdate().
		Model(Mailbox).
		WherePK().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DeleteMailbox(ctx context.Context, id int) error {
	Mailbox := &Mailbox{Id: id}

	_, err := db.Bun.
		NewDelete().
		Model(Mailbox).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	_ = DeleteJobsByRelated(ctx, "mailboxes", id)

	return nil
}
