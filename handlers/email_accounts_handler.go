package handlers

import (
	"app/config"
	"app/errorsx"
	"app/helpers"
	"app/jobs"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages"
	"app/templates/pages/synclist/emailaccounts"
	"app/worker"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
)

func EmailAccountNew(c *echo.Context) error {
	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	return helpers.Render(c, http.StatusOK, emailaccounts.New(emailaccounts.NewProps{
		List: list,
	}))
}

func EmailAccountCreate(c *echo.Context) error {
	var req struct {
		SrcUser     string `form:"SrcUser" validate:"email,required,max=255"`
		SrcPassword string `form:"SrcPassword" validate:"required,max=255"`
		DstUser     string `form:"DstUser" validate:"email,required,max=255"`
		DstPassword string `form:"DstPassword" validate:"required,max=255"`
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		helpers.Retarget(c, "_Error")
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			helpers.Retarget(c, "_Error")
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	err = helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.RenderFragment(c, http.StatusForbidden, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	encryptedSrcPassword, err := helpers.AesEncrypt(req.SrcPassword, config.Config.AppKey)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	encryptedDstPassword, err := helpers.AesEncrypt(req.DstPassword, config.Config.AppKey)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	_, err = models.CreateEmailAccount(c.Request().Context(), list.Id, req.SrcUser, encryptedSrcPassword, req.DstUser, encryptedDstPassword)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.Id))
}

func EmailAccountDelete(c *echo.Context) error {
	listID, err := helpers.ParamAsInt(c, "listID")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), listID)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	account, err := models.FindEmailAccountByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if account.SyncListId != list.Id {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	relatedJobs, err := models.FindJobsByRelated(c.Request().Context(), "email_accounts", account.Id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	for _, job := range relatedJobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			return helpers.Render(c, http.StatusConflict, alert.Error(helpers.MsgErrForbidden))
		}
	}

	err = models.DeleteEmailAccount(c.Request().Context(), id)
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

func EmailAccountDeleteJob(c *echo.Context) error {
	listID, err := helpers.ParamAsInt(c, "listID")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	id, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
	}

	list, err := models.FindSyncListByID(c.Request().Context(), listID)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if list.UserId != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	account, err := models.FindEmailAccountByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if account.SyncListId != list.Id {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	err = models.DeleteJobsByRelated(c.Request().Context(), "email_accounts", account.Id)
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

func EmailAccountJobMigrateStart(c *echo.Context) error {
	listID, err := helpers.ParamAsInt(c, "listID")
	if err != nil {
		slog.Debug("Failed to parse list ID", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}
	accountID, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		slog.Debug("Failed to parse account ID", "error", err)
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	ctx := c.Request().Context()
	userID := helpers.GetUserSessionData(c).ID

	// Fetch list and account in parallel
	listChan := make(chan *models.SyncList, 1)
	accountChan := make(chan *models.EmailAccount, 1)
	errChan := make(chan error, 2)

	go func() {
		list, err := models.FindSyncListByID(ctx, listID)
		errChan <- err
		listChan <- list
	}()

	go func() {
		account, err := models.FindEmailAccountByID(ctx, accountID)
		errChan <- err
		accountChan <- account
	}()

	list := <-listChan
	account := <-accountChan
	if err := <-errChan; err != nil || <-errChan != nil {
		slog.Debug("Failed to fetch list or account", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	// Validate ownership
	if list.UserId != userID || account.SyncListId != list.Id {
		slog.Debug("Invalid ownership", "listID", list.Id, "userID", userID, "accountID", account.Id)
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	// Get existing job
	job, err := models.FindJobByRelated(ctx, "email_accounts", account.Id)
	if err != nil {
		if !errorsx.IsNotFoundError(err) {
			slog.Debug("Failed to find job", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}
	}

	// Handle existing job
	if job.Id != 0 {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			slog.Debug("Job already running or pending", "jobID", job.Id)
			return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
		}

		payload := new(jobs.MigrateAccount)
		err := json.Unmarshal(job.Payload, payload)
		if err != nil {
			slog.Debug("Failed to unmarshal job payload", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		payload.SrcAddr = net.JoinHostPort(list.SrcHost, strconv.Itoa(list.SrcPort))
		payload.DstAddr = net.JoinHostPort(list.DstHost, strconv.Itoa(list.DstPort))

		json, err := json.Marshal(payload)
		if err != nil {
			slog.Debug("Failed to marshal payload", "error", err)
			return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
		}

		job.Payload = json
		job.Status = models.JobStatusPending
		if err := models.UpdateJob(ctx, job); err != nil {
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

	// Create new job
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
		slog.Debug("Failed to marshal payload", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	_, err = models.CreateJobWithRelated(ctx, userID, jobs.MigrateAccountType, "email_accounts", accountID, data)
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

func EmailAccountJobMigrateStop(c *echo.Context) error {
	listID, err := helpers.ParamAsInt(c, "listID")
	if err != nil {
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}
	accountID, err := helpers.ParamAsInt(c, "id")
	if err != nil {
		return helpers.Render(c, http.StatusBadRequest, alert.Error(helpers.MsgErrBadRequest))
	}

	ctx := c.Request().Context()
	userID := helpers.GetUserSessionData(c).ID

	// Fetch list and account in parallel
	listChan := make(chan *models.SyncList, 1)
	accountChan := make(chan *models.EmailAccount, 1)
	errChan := make(chan error, 2)

	go func() {
		list, err := models.FindSyncListByID(ctx, listID)
		errChan <- err
		listChan <- list
	}()

	go func() {
		account, err := models.FindEmailAccountByID(ctx, accountID)
		errChan <- err
		accountChan <- account
	}()

	list := <-listChan
	account := <-accountChan
	if err := <-errChan; err != nil || <-errChan != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	// Validate ownership
	if list.Id != userID || account.SyncListId != list.Id {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	// Get existing job
	job, err := models.FindJobByRelated(ctx, "email_accounts", account.Id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	// Job must exist and be stoppable
	if job == nil || !(job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending) {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	// Get and cancel running job
	runningJob := worker.GetRunningJob(job.Id)
	if runningJob == nil {
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
