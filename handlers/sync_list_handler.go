package handlers

import (
	"app/errorsx"
	"app/helpers"
	"app/jobs"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages/base"
	"app/templates/pages/synclist"
	"app/worker"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

func SyncListIndex(c *echo.Context) error {
	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		slog.Error("Error parsing page parameter", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	syncLists, err := models.FindSyncListsByUserIdPaginated(c.Request().Context(), helpers.GetUserSessionData(c).Id, page)
	if err != nil {
		slog.Error("Error finding sync lists", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	listIds := make([]int, len(syncLists.SyncLists))
	for i, list := range syncLists.SyncLists {
		listIds[i] = list.Id
	}

	statuses, err := models.FindSyncListsStatus(c.Request().Context(), listIds)
	if err != nil {
		slog.Error("Error finding sync list statuses", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	jobStatusMap := make(map[int]models.JobStatus)
	for _, status := range statuses {
		jobStatusMap[status.Id] = status.Status
	}

	return helpers.Render(c, http.StatusOK, synclist.Index(synclist.IndexProps{
		PaginatedSyncLists: syncLists,
		SyncListStatusMap:  jobStatusMap,
	}))
}

func SyncListNew(c *echo.Context) error {
	return helpers.Render(c, http.StatusOK, synclist.New(synclist.NewProps{}))
}

func SyncListCreate(c *echo.Context) error {
	var req struct {
		Name              string `form:"Name" validate:"required,max=255"`
		SrcHost           string `form:"SrcHost" validate:"required,max=255"`
		SrcPort           int    `form:"SrcPort" validate:"required,min=1,max=65535"`
		DstHost           string `form:"DstHost" validate:"required,max=255"`
		DstPort           int    `form:"DstPort" validate:"required,min=1,max=65535"`
		CompareMessageIds bool   `form:"CompareMessageIds" validate:"boolean"`
		CompareLastUid    bool   `form:"CompareLastUid" validate:"boolean"`
	}

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", synclist.New(synclist.NewProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	list, err := models.CreateSyncList(c.Request().Context(), models.CreateSyncListParams{
		UserId:            helpers.GetUserSessionData(c).Id,
		Name:              req.Name,
		SrcHost:           req.SrcHost,
		SrcPort:           req.SrcPort,
		DstHost:           req.DstHost,
		DstPort:           req.DstPort,
		CompareMessageIds: req.CompareMessageIds,
		CompareLastUid:    req.CompareLastUid,
	})
	if err != nil {
		slog.Error("failed to create sync list", "err", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", synclist.New(synclist.NewProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}

func SyncListShow(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Error("failed to parse sync list ID", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		slog.Error("failed to parse page parameter", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	listPaginated, err := models.FindSyncListByIdWithMailboxesPaginated(c.Request().Context(), id, page)
	if err != nil {
		slog.Error("failed to find sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	mailboxIds := make([]int, len(listPaginated.SyncList.Mailboxes))
	for i, mailbox := range listPaginated.SyncList.Mailboxes {
		mailboxIds[i] = mailbox.Id
	}

	jobs, err := models.FindJobsByManyRelated(c.Request().Context(), "mailboxes", mailboxIds)
	if err != nil {
		slog.Error("failed to find jobs", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	mailboxestatusMap := make(map[int]models.JobStatus)
	for _, job := range jobs {
		mailboxestatusMap[*job.RelatedId] = job.Status
	}

	listStatus, err := models.FindSyncListStatus(c.Request().Context(), listPaginated.SyncList.Id)
	if err != nil {
		slog.Error("failed to find sync list status", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	if c.QueryParam("polling") == "true" {
		helpers.Reswap(c, "none")
		return helpers.Render(c, http.StatusOK, synclist.PartShowOOB(synclist.ShowProps{
			SyncList:         listPaginated.SyncList,
			SyncListStatus:   listStatus.Status,
			MailboxStatusMap: mailboxestatusMap,
			PaginatedMailboxes: &models.MailboxesPaginated{
				Mailboxes:  listPaginated.SyncList.Mailboxes,
				Pagination: listPaginated.MailboxPagination,
			},
		}))
	}

	return helpers.Render(c, http.StatusOK, synclist.Show(synclist.ShowProps{
		SyncList:         listPaginated.SyncList,
		SyncListStatus:   listStatus.Status,
		MailboxStatusMap: mailboxestatusMap,
		PaginatedMailboxes: &models.MailboxesPaginated{
			Mailboxes:  listPaginated.SyncList.Mailboxes,
			Pagination: listPaginated.MailboxPagination,
		},
	}))
}

func SyncListEdit(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Error("failed to parse sync list id", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListById(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, base.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		return helpers.Render(c, http.StatusForbidden, base.Error(helpers.MsgErrForbidden))
	}

	return helpers.Render(c, http.StatusOK, synclist.Edit(synclist.EditProps{
		List:   list,
		Values: helpers.StructToValues(list),
	}))
}

func SyncListUpdate(c *echo.Context) error {
	var req struct {
		Name              string `form:"Name" validate:"required,max=255"`
		SrcHost           string `form:"SrcHost" validate:"required,max=255"`
		SrcPort           int    `form:"SrcPort" validate:"required,min=1,max=65535"`
		DstHost           string `form:"DstHost" validate:"required,max=255"`
		DstPort           int    `form:"DstPort" validate:"required,min=1,max=65535"`
		CompareMessageIds bool   `form:"CompareMessageIds" validate:"boolean"`
		CompareLastUid    bool   `form:"CompareLastUid" validate:"boolean"`
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Error("failed to parse sync list id", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByIdWithMailboxes(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("failed to validate sync list", "err", err)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	mailboxIds := make([]int, len(list.Mailboxes))
	for i, mailbox := range list.Mailboxes {
		mailboxIds[i] = mailbox.Id
	}

	relatedJobs, err := models.FindJobsByManyRelated(c.Request().Context(), "mailboxes", mailboxIds)
	if err != nil {
		slog.Error("failed to find jobs by many related", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			slog.Error("failed to validate sync list", "err", "job is running or pending")
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}
	}

	err = helpers.BindAndValidate(c, &req)
	if err != nil {
		slog.Error("failed to bind and validate sync list", "err", err)
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", synclist.Edit(synclist.EditProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	list.Name = req.Name
	list.SrcHost = req.SrcHost
	list.SrcPort = req.SrcPort
	list.DstHost = req.DstHost
	list.DstPort = req.DstPort
	list.CompareMessageIds = req.CompareMessageIds
	list.CompareLastUid = req.CompareLastUid

	err = models.UpdateSyncList(c.Request().Context(), list)
	if err != nil {
		slog.Error("failed to update sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}

func SyncListDelete(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Error("failed to parse sync list ID", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByIdWithMailboxes(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("user is not authorized to delete sync list", "err", err)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	mailboxIds := make([]int, len(list.Mailboxes))
	for i, mailbox := range list.Mailboxes {
		mailboxIds[i] = mailbox.Id
	}

	relatedJobs, err := models.FindJobsByManyRelated(c.Request().Context(), "mailboxes", mailboxIds)
	if err != nil {
		slog.Error("failed to find related jobs", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			slog.Error("job is running or pending", "err", err)
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}
	}

	err = models.DeleteSyncListById(c.Request().Context(), id)
	if err != nil {
		slog.Error("failed to delete sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists")
}

func SyncListDeleteJobs(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Error("failed to parse id", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByIdWithMailboxes(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find sync list", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).Id {
		slog.Error("user is not authorized to delete sync list", "err", err)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	mailboxIds := make([]int, len(list.Mailboxes))
	for i, mailbox := range list.Mailboxes {
		mailboxIds[i] = mailbox.Id
	}

	err = models.DeleteJobsByManyRelated(c.Request().Context(), "mailboxes", mailboxIds)
	if err != nil {
		slog.Error("failed to delete jobs", "err", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
	} else {
		return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id)+"?page="+strconv.Itoa(page))
	}
}

func SyncListJobMigrateStart(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Debug("Invalid ID parameter", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	ctx := c.Request().Context()
	userId := helpers.GetUserSessionData(c).Id

	list, err := models.FindSyncListByIdWithMailboxes(ctx, id)
	if err != nil {
		slog.Debug("Failed to find sync list", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != userId {
		slog.Debug("Unauthorized access attempt", "userID", userId, "listID", id)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	if len(list.Mailboxes) == 0 {
		return c.NoContent(http.StatusOK)
	}

	mailboxIds := make([]int, 0)
	for _, mailbox := range list.Mailboxes {
		mailboxIds = append(mailboxIds, mailbox.Id)
	}

	jobsByMailboxId, err := models.FindJobsByManyRelatedMap(ctx, "mailboxes", mailboxIds)
	if err != nil {
		slog.Debug("Failed to find existing jobs", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	jobsToUpdate := make([]*models.Job, 0)
	newJobPayloads := make([]*json.RawMessage, 0)
	newJobMailboxIds := make([]int, 0)

	for _, mailboxId := range mailboxIds {
		relJobs, exists := jobsByMailboxId[mailboxId]
		if len(relJobs) > 1 {
			slog.Error("Found multiple jobs for mailbox", "mailbox_id", mailboxId)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		if exists {
			relJobs[0].Status = models.JobStatusPending
			now := time.Now()
			relJobs[0].StartedAt = &now
			relJobs[0].FinishedAt = nil
			jobsToUpdate = append(jobsToUpdate, relJobs[0])
		} else {
			payload := jobs.MigrateMailboxPayload{
				SyncListId: list.Id,
				MailboxId:  mailboxId,
			}

			payloadJson, err := json.Marshal(payload)
			if err != nil {
				slog.Debug("Failed to marshal payload", "error", err)
				return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
			}

			newJobPayloads = append(newJobPayloads, (*json.RawMessage)(&payloadJson))
			newJobMailboxIds = append(newJobMailboxIds, mailboxId)
		}
	}

	if len(jobsToUpdate) > 0 {
		if err := models.UpdateJobs(ctx, jobsToUpdate); err != nil {
			slog.Debug("Failed to update jobs", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}
	}

	if len(newJobPayloads) > 0 {
		_, err = models.CreateJobsWithRelated(ctx, userId, jobs.MigrateMailboxType, "mailboxes", newJobMailboxIds, newJobPayloads)
		if err != nil {
			slog.Debug("Failed to create jobs", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}

func SyncListJobMigrateStop(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	ctx := c.Request().Context()
	userId := helpers.GetUserSessionData(c).Id

	list, err := models.FindSyncListByIdWithMailboxes(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}

		slog.Error("Failed to find sync list", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != userId {
		slog.Error("Unauthorized access to sync list", "sync_list_id", list.Id, "user_id", userId)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	if len(list.Mailboxes) == 0 {
		return c.NoContent(http.StatusOK)
	}

	mailboxIds := make([]int, len(list.Mailboxes))
	for i, mailbox := range list.Mailboxes {
		mailboxIds[i] = mailbox.Id
	}

	jobs, err := models.FindJobsByManyRelated(ctx, "mailboxes", mailboxIds)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range jobs {
		job.Status = models.JobStatusInterrupted

		runningJob := worker.GetRunningJob(job.Id)
		if runningJob != nil {
			runningJob.Cancel()
		}
	}

	err = models.UpdateJobs(ctx, jobs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}
