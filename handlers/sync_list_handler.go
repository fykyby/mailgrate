package handlers

import (
	"app/errorsx"
	"app/helpers"
	"app/jobs"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages"
	"app/templates/pages/synclist"
	"app/worker"
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
)

func SyncListIndex(c *echo.Context) error {
	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		slog.Error("Error parsing page parameter", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
	}

	syncLists, err := models.FindSyncListsByUserIDPaginated(c.Request().Context(), helpers.GetUserSessionData(c).ID, page)
	if err != nil {
		slog.Error("Error finding sync lists", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
	}

	listIDs := make([]int, len(syncLists.SyncLists))
	for i, list := range syncLists.SyncLists {
		listIDs[i] = list.Id
	}

	statuses, err := models.FindSyncListsStatus(c.Request().Context(), listIDs)
	if err != nil {
		slog.Error("Error finding sync list statuses", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
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
		UserId:            helpers.GetUserSessionData(c).ID,
		Name:              req.Name,
		SrcHost:           req.SrcHost,
		SrcPort:           req.SrcPort,
		DstHost:           req.DstHost,
		DstPort:           req.DstPort,
		CompareMessageIds: req.CompareMessageIds,
		CompareLastUid:    req.CompareLastUid,
	})
	if err != nil {
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
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListIDPaginated(c.Request().Context(), list.Id, page)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountIDs := make([]int, len(accounts.EmailAccounts))
	for i, account := range accounts.EmailAccounts {
		accountIDs[i] = account.Id
	}

	jobs, err := models.FindJobsByRelatedBulk(c.Request().Context(), "email_accounts", accountIDs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountStatusMap := make(map[int]models.JobStatus)
	for _, job := range jobs {
		accountStatusMap[job.RelatedId] = job.Status
	}

	listStatus, err := models.FindSyncListStatus(c.Request().Context(), list.Id)
	if err != nil {
		log.Println(err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Render(c, http.StatusOK, synclist.Show(synclist.ShowProps{
		SyncList:               list,
		SyncListStatus:         listStatus.Status,
		EmailAccountStatusMap:  accountStatusMap,
		PaginatedEmailAccounts: accounts,
	}))
}

func SyncListEdit(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
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
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	err = helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", synclist.Edit(synclist.EditProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListID(c.Request().Context(), list.Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountIDs := make([]int, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.Id
	}

	relatedJobs, err := models.FindJobsByRelatedBulk(c.Request().Context(), "email_accounts", accountIDs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}

		payload := new(jobs.MigrateAccount)
		err = json.Unmarshal(job.Payload, payload)
		if err != nil {
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		payload.SrcAddr = net.JoinHostPort(list.SrcHost, strconv.Itoa(list.SrcPort))
		payload.DstAddr = net.JoinHostPort(list.DstHost, strconv.Itoa(list.DstPort))
		payload.CompareMessageIds = req.CompareMessageIds
		payload.CompareLastUid = req.CompareLastUid

		json, err := json.Marshal(payload)
		if err != nil {
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		job.Payload = json
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
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	err = models.UpdateJobs(c.Request().Context(), relatedJobs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}

func SyncListDelete(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListID(c.Request().Context(), list.Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountIDs := make([]int, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.Id
	}

	relatedJobs, err := models.FindJobsByRelatedBulk(c.Request().Context(), "email_accounts", accountIDs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}
	}

	err = models.DeleteSyncListByID(c.Request().Context(), id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists")
}

func SyncListDeleteJobs(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListID(c.Request().Context(), list.Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountIDs := make([]int, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.Id
	}

	err = models.DeleteJobsByRelatedBulk(c.Request().Context(), "email_accounts", accountIDs)
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

func SyncListJobMigrateStart(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Debug("Invalid ID parameter", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	ctx := c.Request().Context()
	userID := helpers.GetUserSessionData(c).ID

	list, err := models.FindSyncListByID(ctx, id)
	if err != nil {
		slog.Debug("Failed to find sync list", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != userID {
		slog.Debug("Unauthorized access attempt", "userID", userID, "listID", id)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListID(ctx, list.Id)
	if err != nil {
		slog.Debug("Failed to find email accounts", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if len(accounts) == 0 {
		return c.NoContent(http.StatusOK)
	}

	// Extract account IDs
	accountIDs := make([]int, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.Id
	}

	// Fetch existing jobs
	existingJobs, err := models.FindJobsByRelatedBulk(ctx, "email_accounts", accountIDs)
	if err != nil {
		slog.Debug("Failed to find existing jobs", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	// Validate and organize jobs by account
	jobsByAccountID := make(map[int]*models.Job)
	for _, job := range existingJobs {
		if _, exists := jobsByAccountID[job.RelatedId]; exists {
			slog.Error("multiple jobs found for email account", "email_account", job.RelatedId)
			return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
		}
		jobsByAccountID[job.RelatedId] = job
	}

	// Update existing jobs and collect new job payloads
	jobsToUpdate := make([]*models.Job, 0)
	newJobPayloads := make([]json.RawMessage, 0)
	newJobAccountIDs := make([]int, 0)

	for _, account := range accounts {
		job, exists := jobsByAccountID[account.Id]
		if exists {
			// Update existing job if not already running/pending
			if !(job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending) {
				payload := new(jobs.MigrateAccount)
				err := json.Unmarshal(job.Payload, payload)
				if err != nil {
					slog.Debug("Failed to unmarshal job payload", "error", err)
					return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
				}

				payload.SrcAddr = net.JoinHostPort(list.SrcHost, strconv.Itoa(list.SrcPort))
				payload.DstAddr = net.JoinHostPort(list.DstHost, strconv.Itoa(list.DstPort))
				payload.CompareMessageIds = list.CompareMessageIds

				json, err := json.Marshal(payload)
				if err != nil {
					slog.Debug("Failed to marshal job payload", "error", err)
					return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
				}

				job.Payload = json
				job.Status = models.JobStatusPending
				jobsToUpdate = append(jobsToUpdate, job)
			}
		} else {
			payload := jobs.NewMigrateAccount(jobs.NewMigrateAccountParams{
				SrcAddr:           net.JoinHostPort(list.SrcHost, strconv.Itoa(list.SrcPort)),
				DstAddr:           net.JoinHostPort(list.DstHost, strconv.Itoa(list.DstPort)),
				SrcUser:           account.SrcUser,
				SrcPassword:       account.SrcPasswordHash,
				DstUser:           account.DstUser,
				DstPassword:       account.DstPasswordHash,
				CompareMessageIDs: list.CompareMessageIds,
			})

			data, err := json.Marshal(payload)
			if err != nil {
				slog.Debug("Failed to marshal job payload", "error", err)
				return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
			}
			newJobPayloads = append(newJobPayloads, data)
			newJobAccountIDs = append(newJobAccountIDs, account.Id)
		}
	}

	// Update existing jobs
	if len(jobsToUpdate) > 0 {
		if err := models.UpdateJobs(ctx, jobsToUpdate); err != nil {
			slog.Debug("Failed to update jobs", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}
	}

	// Create new jobs
	if len(newJobPayloads) > 0 {
		_, err = models.CreateJobsWithRelated(ctx, userID, jobs.MigrateAccountType, newJobPayloads, "email_accounts", newJobAccountIDs)
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
	userID := helpers.GetUserSessionData(c).ID

	list, err := models.FindSyncListByID(ctx, id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != userID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListID(ctx, list.Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if len(accounts) == 0 {
		return c.NoContent(http.StatusOK)
	}

	// Extract account IDs
	accountIDs := make([]int, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.Id
	}

	// Fetch and cancel all running jobs
	jobs, err := models.FindJobsByRelatedBulk(ctx, "email_accounts", accountIDs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range jobs {
		runningJob := worker.GetRunningJob(job.Id)
		if runningJob != nil {
			runningJob.Cancel()
		}
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}
