package handlers

import (
	"app/httpx"
	"app/templates/pages/app"
	"net/http"

	"github.com/labstack/echo/v5"
)

func DashboardShow(c *echo.Context) error {
	return httpx.Render(c, http.StatusOK, app.Dashboard())
}
