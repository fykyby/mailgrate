package handlers

import (
	"app/helpers"
	"app/templates/pages/app"
	"net/http"

	"github.com/labstack/echo/v5"
)

func DashboardShow(c *echo.Context) error {
	return helpers.Render(c, http.StatusOK, app.Dashboard())
}
