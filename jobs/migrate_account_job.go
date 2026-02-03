package jobs

import (
	"app/models"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var MigrateAccountType models.JobType = "migrate_account"

type MigrateAccount struct {
	SyncListID        int
	EmailAccountID    int
	Source            string
	Destination       string
	Login             string
	Password          string
	FolderLastUid     map[string]uint32
	FolderUidValidity map[string]uint32
}

func NewMigrateAccount(syncListID int, emailAccountID int, src string, dst string, login string, password string) *MigrateAccount {
	return &MigrateAccount{
		EmailAccountID:    emailAccountID,
		Source:            src,
		Destination:       dst,
		Login:             login,
		Password:          password,
		FolderLastUid:     make(map[string]uint32),
		FolderUidValidity: make(map[string]uint32),
	}
}

func (j *MigrateAccount) Run(ctx context.Context) (err error) {
	defer func() {
		switch {
		case errors.Is(err, context.Canceled):
			_ = models.UpdateEmailAccountStatus(ctx, j.EmailAccountID, models.JobStatusInterrupted)
		case err != nil:
			_ = models.UpdateEmailAccountStatus(ctx, j.EmailAccountID, models.JobStatusFailed)
		default:
			_ = models.UpdateEmailAccountStatus(ctx, j.EmailAccountID, models.JobStatusCompleted)
		}
	}()

	_ = models.UpdateEmailAccountStatus(ctx, j.EmailAccountID, models.JobStatusRunning)

	sourceClient, err := client.DialTLS(j.Source, nil)
	if err != nil {
		return err
	}
	defer sourceClient.Close()

	destClient, err := client.DialTLS(j.Destination, nil)
	if err != nil {
		return err
	}
	defer destClient.Close()

	if err := sourceClient.Login(j.Login, j.Password); err != nil {
		return err
	}

	if err := destClient.Login(j.Login, j.Password); err != nil {
		return err
	}

	foldersChan := make(chan *imap.MailboxInfo)
	listFoldersDone := make(chan error, 1)
	go func() {
		listFoldersDone <- sourceClient.List("", "*", foldersChan)
	}()

	folderNames := []string{}
	for mbox := range foldersChan {
		folderNames = append(folderNames, mbox.Name)
	}

	err = <-listFoldersDone
	if err != nil {
		return err
	}

	for _, folderName := range folderNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		srcFolder, err := sourceClient.Select(folderName, true)
		if err != nil {
			return err
		}

		if j.FolderUidValidity[folderName] == 0 || j.FolderUidValidity[folderName] != srcFolder.UidValidity {
			j.FolderUidValidity[folderName] = srcFolder.UidValidity
			j.FolderLastUid[folderName] = 0
		}

		criteria := imap.NewSearchCriteria()
		criteria.Uid = &imap.SeqSet{}
		criteria.Uid.AddRange(j.FolderLastUid[folderName], 4294967295)
		uids, err := sourceClient.Search(criteria)
		if err != nil {
			return err
		}

		if len(uids) == 0 {
			continue
		}

		if err := destClient.Create(folderName); err != nil {
			if !strings.Contains(strings.ToUpper(err.Error()), "ALREADYEXISTS") && !strings.Contains(strings.ToUpper(err.Error()), "ALREADY EXISTS") {
				return err
			}
		}

		seqset := &imap.SeqSet{}
		seqset.AddNum(uids...)

		messages := make(chan *imap.Message)
		fetchMessagesDone := make(chan error, 1)
		go func() {
			fetchMessagesDone <- sourceClient.Fetch(seqset, []imap.FetchItem{
				imap.FetchEnvelope,
				imap.FetchFlags,
				imap.FetchRFC822,
				imap.FetchUid,
			}, messages)
		}()

		for msg := range messages {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if msg.Uid < j.FolderLastUid[folderName] {
				continue
			}

			literal := msg.GetBody(&imap.BodySectionName{})
			if literal == nil {
				continue
			}

			flags := msg.Flags
			date := msg.Envelope.Date
			uid := msg.Uid

			appendDone := make(chan error, 1)
			go func(lit imap.Literal, f []string, d time.Time, u uint32) {
				select {
				case appendDone <- destClient.Append(folderName, f, d, lit):
				case <-ctx.Done():
				}
			}(literal, flags, date, uid)

			select {
			case err := <-appendDone:
				if err != nil {
					return err
				}
				j.FolderLastUid[folderName] = uid
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		select {
		case err := <-fetchMessagesDone:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
