package handlers

import (
	"app/config"
	"app/errorsx"
	"app/helpers"
	"app/jobs"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages/base"
	"app/templates/pages/synclist/mailbox"
	"app/worker"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

func MailboxNew(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Error("failed to parse id", "err", err.Error())
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListById(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find sync list", "err", err.Error())
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("user not authorized")
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	return helpers.Render(c, http.StatusOK, mailbox.New(mailbox.NewProps{
		List: list,
	}))
}

func MailboxCreate(c *echo.Context) error {
	var req struct {
		SrcUser     string `form:"SrcUser" validate:"email,required,max=255"`
		SrcPassword string `form:"SrcPassword" validate:"required,max=255"`
		DstUser     string `form:"DstUser" validate:"email,required,max=255"`
		DstPassword string `form:"DstPassword" validate:"required,max=255"`
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		helpers.Retarget(c, "_Error")
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListById(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			helpers.Retarget(c, "_Error")
			return helpers.Render(c, http.StatusNotFound, base.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find sync list", "err", err.Error())
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	err = helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", mailbox.New(mailbox.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("user is not authorized to access this sync list")
		return helpers.RenderFragment(c, http.StatusForbidden, "form", mailbox.New(mailbox.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	relatedJobs, err := models.FindJobsByRelated(c.Request().Context(), "mailboxes", list.Mailboxes[0].Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}
	}

	encryptedSrcPassword, err := helpers.AesEncrypt(req.SrcPassword, config.Config.AppKey)
	if err != nil {
		slog.Error("failed to encrypt source password", "err", err.Error())
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", mailbox.New(mailbox.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	encryptedDstPassword, err := helpers.AesEncrypt(req.DstPassword, config.Config.AppKey)
	if err != nil {
		slog.Error("failed to encrypt destination password", "err", err.Error())
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", mailbox.New(mailbox.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	_, err = models.CreateMailbox(c.Request().Context(), list.Id, req.SrcUser, encryptedSrcPassword, req.DstUser, encryptedDstPassword)
	if err != nil {
		slog.Error("failed to create mailbox", "err", err.Error())
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", mailbox.New(mailbox.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}

func MailboxDelete(c *echo.Context) error {
	listId, err := helpers.ParamAsInt(c, "listId")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	list, err := models.FindSyncListByIdWithMailboxById(c.Request().Context(), listId, id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("Failed to find sync list with mailbox", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("User is not authorized to access this sync list", "userId", helpers.GetUserSessionData(c).Id, "syncListId", list.Id)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	if len(list.Mailboxes) == 0 {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
	}

	relatedJobs, err := models.FindJobsByRelated(c.Request().Context(), "mailboxes", list.Mailboxes[0].Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}
	}

	err = models.DeleteMailbox(c.Request().Context(), list.Mailboxes[0].Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
	} else {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id)+"?page="+strconv.Itoa(page))
	}
}

func MailboxDeleteJob(c *echo.Context) error {
	listId, err := helpers.ParamAsInt(c, "listId")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	list, err := models.FindSyncListByIdWithMailboxById(c.Request().Context(), listId, id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("Failed to find sync list with mailbox", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("Unauthorized access to sync list", "listId", listId, "userId", helpers.GetUserSessionData(c).Id)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	err = models.DeleteJobsByRelated(c.Request().Context(), "mailboxes", list.Mailboxes[0].Id)
	if err != nil {
		slog.Error("Failed to delete jobs", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
	} else {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id)+"?page="+strconv.Itoa(page))
	}
}

func MailboxJobMigrateStart(c *echo.Context) error {
	listId, err := helpers.ParamAsInt(c, "listId")
	if err != nil {
		slog.Debug("Failed to parse list ID", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}
	mailboxId, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Debug("Failed to parse account ID", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	userId := helpers.GetUserSessionData(c).Id

	list, err := models.FindSyncListByIdWithMailboxById(c.Request().Context(), listId, mailboxId)
	if err != nil {
		if !errorsx.IsNotFoundError(err) {
			slog.Debug("Failed to find sync list", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		slog.Debug("Failed to find sync list", "error", err)
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Debug("User does not own the sync list", "userID", userId, "listID", listId)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	if len(list.Mailboxes) == 0 {
		slog.Debug("Sync list has no mailboxes", "listID", listId)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	job, err := models.FindJobByRelated(c.Request().Context(), "mailboxes", mailboxId)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Debug("Failed to find job", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if job != nil {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			slog.Debug("Job already running or pending", "jobID", job.Id)
			return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
		}

		job.Status = models.JobStatusPending
		now := time.Now()
		job.StartedAt = &now

		err = models.UpdateJob(c.Request().Context(), job)
		if err != nil {
			slog.Debug("Failed to update job", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		page, err := helpers.QueryParamAsInt(c, "page")
		if err != nil {
			return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
		} else {
			return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id)+"?page="+strconv.Itoa(page))
		}
	}

	payload := jobs.MigrateMailboxPayload{
		SyncListId: list.Id,
		MailboxId:  mailboxId,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		slog.Debug("Failed to marshal payload", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	_, err = models.CreateJobWithRelated(c.Request().Context(), userId, jobs.MigrateMailboxType, "mailboxes", mailboxId, (*json.RawMessage)(&payloadJson))
	if err != nil {
		slog.Debug("Failed to create job", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}
	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
	} else {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id)+"?page="+strconv.Itoa(page))
	}

}

func MailboxJobMigrateStop(c *echo.Context) error {
	listId, err := helpers.ParamAsInt(c, "listId")
	if err != nil {
		slog.Debug("Failed to parse list ID", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}
	mailboxId, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Debug("Failed to parse account ID", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	list, err := models.FindSyncListByIdWithMailboxById(c.Request().Context(), listId, mailboxId)
	if err != nil {
		if !errorsx.IsNotFoundError(err) {
			slog.Debug("Failed to find sync list", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		slog.Debug("Failed to find sync list", "error", err)
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Debug("User does not own the sync list", "userID", helpers.GetUserSessionData(c).Id, "listID", listId)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	if len(list.Mailboxes) == 0 {
		slog.Debug("Sync list has no mailboxes", "listID", listId)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	job, err := models.FindJobByRelated(c.Request().Context(), "mailboxes", mailboxId)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Debug("Failed to find job", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if job == nil || !(job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending) {
		slog.Debug("Job not found or not running")
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	job.Status = models.JobStatusInterrupted
	err = models.UpdateJob(c.Request().Context(), job)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	runningJob := worker.GetRunningJob(job.Id)
	if runningJob == nil {
		slog.Debug("Running job not found")
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	runningJob.Cancel()

	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
	} else {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id)+"?page="+strconv.Itoa(page))
	}
}
