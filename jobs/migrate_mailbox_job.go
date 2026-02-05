package jobs

import (
	"app/config"
	"app/helpers"
	"app/models"
	"app/worker"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var MigrateMailboxType models.JobType = "migrate_account"

type MigrateMailbox struct {
	SyncList *models.SyncList
	Mailbox  *models.Mailbox
}

type MigrateMailboxPayload struct {
	SyncListId int `json:"syncListId"`
	MailboxId  int `json:"mailboxId"`
}

func MigrateMailboxFactory(ctx context.Context, payload *json.RawMessage) (worker.JobHandler, error) {
	migrateMailboxPayload := new(MigrateMailboxPayload)

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

	handler := &MigrateMailbox{
		SyncList: list,
		Mailbox:  mailbox,
	}

	return handler, nil
}

func (j *MigrateMailbox) Run(ctx context.Context) (err error) {
	slog.Debug("Starting migration")

	srcAddr := net.JoinHostPort(j.SyncList.SrcHost, strconv.Itoa(j.SyncList.SrcPort))
	dstAddr := net.JoinHostPort(j.SyncList.DstHost, strconv.Itoa(j.SyncList.DstPort))

	var srcClient *client.Client
	var dstClient *client.Client
	var dialWg sync.WaitGroup
	var srcClientErr error
	var dstClientErr error

	dialWg.Add(2)
	go func() {
		defer dialWg.Done()
		srcClient, srcClientErr = client.DialTLS(srcAddr, nil)
		if srcClient == nil && srcClientErr == nil {
			srcClientErr = errors.New("source client is nil")
		}
	}()
	go func() {
		defer dialWg.Done()
		dstClient, dstClientErr = client.DialTLS(dstAddr, nil)
		if dstClient == nil && dstClientErr == nil {
			dstClientErr = errors.New("destination client is nil")
		}
	}()

	done := make(chan struct{})
	go func() {
		dialWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if srcClientErr != nil {
			slog.Debug("Failed to connect to source server (TLS)", "error", srcClientErr)
			srcClient, err = client.Dial(srcAddr)
			if err != nil {
				slog.Debug("Failed to connect to source server (no TLS)", "error", err)
				return err
			}

			err = srcClient.StartTLS(&tls.Config{
				InsecureSkipVerify: config.Config.Debug,
			})
			if err != nil {
				slog.Debug("Failed to start source TLS", "error", err)
				return err
			}

			slog.Debug("Connected to source server")
		}

		if dstClientErr != nil {
			slog.Debug("Failed to connect to destination server (TLS)", "error", dstClientErr)
			dstClient, err = client.Dial(dstAddr)
			if err != nil {
				slog.Debug("Failed to connect to destination server (no TLS)", "error", err)
				return err
			}

			err = dstClient.StartTLS(&tls.Config{
				InsecureSkipVerify: config.Config.Debug,
			})
			if err != nil {
				slog.Debug("Failed to start destination TLS", "error", err)
				return err
			}

			slog.Debug("Connected to destination server")
		}

		if srcClient != nil {
			defer srcClient.Logout()
		}
		if dstClient != nil {
			defer dstClient.Logout()
		}

	case <-time.After(15 * time.Second):
		return errors.New("dial timeout")
	}

	decryptedSrcPassword, err := helpers.AesDecrypt(j.Mailbox.SrcPasswordHash, config.Config.AppKey)
	if err != nil {
		slog.Debug("Failed to decrypt source password", "error", err)
		return err
	}

	decryptedDstPassword, err := helpers.AesDecrypt(j.Mailbox.DstPasswordHash, config.Config.AppKey)
	if err != nil {
		slog.Debug("Failed to decrypt destination password", "error", err)
		return err
	}

	if err := srcClient.Login(j.Mailbox.SrcUser, decryptedSrcPassword); err != nil {
		slog.Debug("Failed to login to source account", "error", err)
		return err
	}

	if err := dstClient.Login(j.Mailbox.DstUser, decryptedDstPassword); err != nil {
		slog.Debug("Failed to login to destination account", "error", err)
		return err
	}

	foldersChan := make(chan *imap.MailboxInfo)
	listFoldersDone := make(chan error, 1)
	go func() {
		listFoldersDone <- srcClient.List("", "*", foldersChan)
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

		srcFolder, err := srcClient.Select(folderName, true)
		if err != nil {
			slog.Debug("Failed to select source folder", "folder", folderName, "error", err)
			return err
		}

		if j.Mailbox.FolderUidValidity[folderName] == 0 || j.Mailbox.FolderUidValidity[folderName] != srcFolder.UidValidity {
			j.Mailbox.FolderUidValidity[folderName] = srcFolder.UidValidity
			j.Mailbox.FolderLastUid[folderName] = 0
		}

		criteria := imap.NewSearchCriteria()
		if j.SyncList.CompareLastUid {
			criteria.Uid = &imap.SeqSet{}
			criteria.Uid.AddRange(j.Mailbox.FolderLastUid[folderName]+1, 4294967295)
		}

		uids, err := srcClient.Search(criteria)
		if err != nil {
			slog.Debug("Failed to search for messages", "connection", "source", "folder", folderName, "error", err)
			return err
		}

		if len(uids) == 0 {
			continue
		}

		if err := dstClient.Create(folderName); err != nil {
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
			fetchMessagesDone <- srcClient.Fetch(seqset, []imap.FetchItem{
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

			if j.SyncList.CompareLastUid && msg.Uid <= j.Mailbox.FolderLastUid[folderName] {
				slog.Debug("Message Uid is less than or equal to last UID", "messageID", msg.Envelope.MessageId)
				continue
			}

			if j.SyncList.CompareMessageIds {
				dstCriteria := imap.NewSearchCriteria()
				dstCriteria.Header.Set("Message-ID", msg.Envelope.MessageId)
				dstCriteria.WithoutFlags = []string{"\\Deleted"}

				_, err = dstClient.Select(folderName, true)
				if err != nil {
					slog.Debug("Failed to select source folder", "folder", folderName, "error", err)
					continue
				}

				existing, err := dstClient.Search(dstCriteria)
				if err != nil {
					slog.Debug("Failed to search for messages", "connection", "destination", "folder", folderName, "error", err)
					continue
				}

				if len(existing) > 0 {
					slog.Debug("Message-ID already exists in destination", "messageID", msg.Envelope.MessageId)
					continue
				}
			}

			slog.Debug("Migrating message", "messageID", msg.Envelope.MessageId)

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
				case appendDone <- dstClient.Append(folderName, f, d, lit):
				case <-ctx.Done():
				}
			}(literal, flags, date, uid)

			select {
			case err := <-appendDone:
				if err != nil {
					return err
				}
				j.Mailbox.FolderLastUid[folderName] = uid
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

func (j *MigrateMailbox) OnStop(ctx context.Context) error {
	err := models.UpdateMailbox(ctx, j.Mailbox)
	if err != nil {
		return err
	}

	return nil
}
