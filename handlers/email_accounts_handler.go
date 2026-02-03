package handlers

import (
	"app/config"
	"app/errorsx"
	"app/helpers"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages"
	"app/templates/pages/synclist/emailaccounts"
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	return helpers.Render(c, http.StatusOK, emailaccounts.New(emailaccounts.NewProps{
		List: list,
	}))
}

func EmailAccountCreate(c *echo.Context) error {
	var req struct {
		Login    string `form:"Login" validate:"email,required,max=255"`
		Password string `form:"Password" validate:"required,max=255"`
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.RenderFragment(c, http.StatusForbidden, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	encryptedPassword, err := helpers.AesEncrypt(req.Password, config.Config.AppKey)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	_, err = models.CreateEmailAccount(c.Request().Context(), list.ID, req.Login, encryptedPassword)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", emailaccounts.New(emailaccounts.NewProps{
			List:   list,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.ID))
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

	if list.UserID != helpers.GetUserSessionData(c).ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	account, err := models.FindEmailAccountByID(c.Request().Context(), id)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, alert.Error(helpers.MsgErrNotFound))
		}
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	if account.SyncListID != list.ID {
		return helpers.Render(c, http.StatusForbidden, alert.Error(helpers.MsgErrForbidden))
	}

	err = models.DeleteEmailAccount(c.Request().Context(), id)
	if err != nil {
		return helpers.Render(c, http.StatusInternalServerError, alert.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/app/sync-lists/"+strconv.Itoa(list.ID))
}

func EmailAccountJobMigrateStart(c *echo.Context) error {
	return c.String(200, "ok")
}

func EmailAccountJobMigrateStop(c *echo.Context) error {
	return c.String(200, "ok")
}
