package app

import (
	"app/handlers"
	"app/helpers"
	"app/templates/pages"
	"net/http"

	middlewarex "app/middleware"

	"github.com/labstack/echo/v5"
)

func RegisterRoutes(e *echo.Echo) {
	aa := e.Group("")
	aa.Use(middlewarex.WithAuthAny)
	aa.GET("/", func(c *echo.Context) error {
		return helpers.Render(c, http.StatusOK, pages.Home())
	})
	aa.GET("/password-reset", handlers.UserShowRequestPasswordReset)
	aa.POST("/password-reset", handlers.UserRequestPasswordReset)
	aa.GET("/password-reset/:token", handlers.UserShowPasswordReset)
	aa.POST("/password-reset/:token", handlers.UserPasswordReset)
	aa.GET("/contact", handlers.ContactShow)
	aa.POST("/contact", handlers.ContactSend)

	af := e.Group("")
	af.Use(middlewarex.WithAuthForbidden)
	af.GET("/sign-up", handlers.UserShowSignUp)
	af.POST("/sign-up", handlers.UserSignUp)
	af.GET("/sign-up/:token", handlers.UserSignUpConfirm)
	af.GET("/log-in", handlers.UserShowLogIn)
	af.POST("/log-in", handlers.UserLogIn)

	ar := e.Group("")
	ar.Use(middlewarex.WithAuthRequired)
	ar.POST("/log-out", handlers.UserLogOut)
	ar.GET("/app", handlers.DashboardShow)
}
