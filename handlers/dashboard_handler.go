package handlers

import (
	"app/helpers"

	"github.com/labstack/echo/v5"
)

func DashboardShow(c *echo.Context) error {
	return helpers.Redirect(c, "/app/sync-lists")
	// return helpers.Render(c, http.StatusOK, app.Dashboard())
}
