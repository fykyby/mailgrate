package jobs

import (
	"app/config"
	"app/helpers"
	"app/models"
	"context"
	"crypto/tls"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var MigrateAccountType models.JobType = "migrate_account"

type MigrateAccount struct {
	Source            string
	Destination       string
	SrcUser           string
	SrcPassword       string
	DstUser           string
	DstPassword       string
	FolderLastUid     map[string]uint32
	FolderUidValidity map[string]uint32
}

func NewMigrateAccount(syncListID int, emailAccountID int, src string, dst string, srcUser string, srcPassword string, dstUser string, dstPassword string) *MigrateAccount {
	return &MigrateAccount{
		Source:            src,
		Destination:       dst,
		SrcUser:           srcUser,
		SrcPassword:       srcPassword,
		DstUser:           dstUser,
		DstPassword:       dstPassword,
		FolderLastUid:     make(map[string]uint32),
		FolderUidValidity: make(map[string]uint32),
	}
}

func (j *MigrateAccount) Run(ctx context.Context) (err error) {
	sourceClient, err := client.Dial(j.Source)
	if err != nil {
		slog.Debug("Failed to connect to source server", "error", err)
		return err
	}
	defer sourceClient.Logout()

	err = sourceClient.StartTLS(&tls.Config{
		InsecureSkipVerify: config.Config.IsDev,
	})
	if err != nil {
		slog.Debug("Failed to start source TLS", "error", err)
		return err
	}

	destClient, err := client.Dial(j.Destination)
	if err != nil {
		slog.Debug("Failed to connect to destination server", "error", err)
		return err
	}
	defer destClient.Logout()

	err = destClient.StartTLS(&tls.Config{
		InsecureSkipVerify: config.Config.IsDev,
	})
	if err != nil {
		slog.Debug("Failed to start destination TLS", "error", err)
		return err
	}

	decryptedSrcPassword, err := helpers.AesDecrypt(j.SrcPassword, config.Config.AppKey)
	if err != nil {
		slog.Debug("Failed to decrypt source password", "error", err)
		return err
	}

	decryptedDstPassword, err := helpers.AesDecrypt(j.DstPassword, config.Config.AppKey)
	if err != nil {
		slog.Debug("Failed to decrypt destination password", "error", err)
		return err
	}

	if err := sourceClient.Login(j.SrcUser, decryptedSrcPassword); err != nil {
		slog.Debug("Failed to login to source account", "error", err)
		return err
	}

	if err := destClient.Login(j.DstUser, decryptedDstPassword); err != nil {
		slog.Debug("Failed to login to destination account", "error", err)
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
		slog.Debug("Failed to list folders", "error", err)
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
			slog.Debug("Failed to select source folder", "folder", folderName, "error", err)
			return err
		}

		if j.FolderUidValidity[folderName] == 0 || j.FolderUidValidity[folderName] != srcFolder.UidValidity {
			j.FolderUidValidity[folderName] = srcFolder.UidValidity
			j.FolderLastUid[folderName] = 0
		}

		criteria := imap.NewSearchCriteria()
		criteria.Uid = &imap.SeqSet{}
		criteria.Uid.AddRange(j.FolderLastUid[folderName]+1, 4294967295)
		uids, err := sourceClient.Search(criteria)
		if err != nil {
			slog.Debug("Failed to search for messages", "folder", folderName, "error", err)
			return err
		}

		if len(uids) == 0 {
			continue
		}

		if err := destClient.Create(folderName); err != nil {
			if !strings.Contains(strings.ToUpper(err.Error()), "ALREADYEXISTS") && !strings.Contains(strings.ToUpper(err.Error()), "ALREADY EXISTS") {
				slog.Debug("Failed to create destination folder", "folder", folderName, "error", err)
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

			slog.Debug("Fetched message", "uid", msg.Uid)

			if msg.Uid <= j.FolderLastUid[folderName] {
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
				slog.Debug("Failed to fetch messages", "folder", folderName, "error", err)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
