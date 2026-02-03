package handlers

import (
	"app/errorsx"
	"app/helpers"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages"
	"app/templates/pages/synclist"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
)

func SyncListIndex(c *echo.Context) error {
	page, err := helpers.QueryParamAsInt(c, "page")
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
	}

	syncLists, err := models.FindSyncListsByUserIDPaginated(c.Request().Context(), helpers.GetUserSessionData(c).ID, page)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, pages.Error(helpers.MsgErrGeneric))
	}

	listIDs := make([]int, len(syncLists.SyncLists))
	for i, list := range syncLists.SyncLists {
		listIDs[i] = list.ID
	}

	statuses, err := models.FindSyncListsStatus(c.Request().Context(), listIDs)
	if err != nil {
		log.Printf("Error finding sync list statuses: %v", err)
		return err
	}

	jobStatusMap := make(map[int]models.JobStatus)
	for _, status := range statuses {
		jobStatusMap[status.ID] = status.Status
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
		Name            string `form:"Name" validate:"required,max=255"`
		SourceHost      string `form:"SourceHost" validate:"required,max=255"`
		SourcePort      int    `form:"SourcePort" validate:"required,min=1,max=65535"`
		DestinationHost string `form:"DestinationHost" validate:"required,max=255"`
		DestinationPort int    `form:"DestinationPort" validate:"required,min=1,max=65535"`
	}

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", synclist.New(synclist.NewProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	list, err := models.CreateSyncList(c.Request().Context(), helpers.GetUserSessionData(c).ID, req.Name, req.SourceHost, req.SourcePort, req.DestinationHost, req.DestinationPort)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", synclist.New(synclist.NewProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.ID))
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	accounts, err := models.FindEmailAccountsBySyncListIDPaginated(c.Request().Context(), list.ID, page)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountIDs := make([]int, len(accounts.EmailAccounts))
	for i, account := range accounts.EmailAccounts {
		accountIDs[i] = account.ID
	}

	jobs, err := models.FindJobsByRelatedMany(c.Request().Context(), "email_accounts", accountIDs)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	accountStatusMap := make(map[int]models.JobStatus)
	for _, job := range jobs {
		accountStatusMap[job.RelatedID] = job.Status
	}

	listStatus, err := models.FindSyncListStatus(c.Request().Context(), list.ID)
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	return helpers.Render(c, http.StatusOK, synclist.Edit(synclist.EditProps{
		List:   list,
		Values: helpers.StructToValues(list),
	}))
}

func SyncListUpdate(c *echo.Context) error {
	var req struct {
		Name            string `form:"Name" validate:"required,max=255"`
		SourceHost      string `form:"SourceHost" validate:"required,max=255"`
		SourcePort      int    `form:"SourcePort" validate:"required,min=1,max=65535"`
		DestinationHost string `form:"DestinationHost" validate:"required,max=255"`
		DestinationPort int    `form:"DestinationPort" validate:"required,min=1,max=65535"`
	}

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", synclist.New(synclist.NewProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	list.Name = req.Name
	list.SourceHost = req.SourceHost
	list.SourcePort = req.SourcePort
	list.DestinationHost = req.DestinationHost
	list.DestinationPort = req.DestinationPort

	err = models.UpdateSyncList(c.Request().Context(), list)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.ID))
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	err = models.DeleteSyncListByID(c.Request().Context(), id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists")
}

func SyncListJobMigrateStart(c *echo.Context) error {
	return c.String(200, "ok")
}

func SyncListJobMigrateStop(c *echo.Context) error {
	return c.String(200, "ok")
}
