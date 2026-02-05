package app

import (
	"app/handlers"
	"app/helpers"
	"app/templates/pages/base"
	"net/http"

	middlewarex "app/middleware"

	"github.com/labstack/echo/v5"
)

func RegisterRoutes(e *echo.Echo) {
	aa := e.Group("")
	aa.Use(middlewarex.WithAuthAny)
	aa.GET("/", func(c *echo.Context) error {
		return helpers.Render(c, http.StatusOK, base.Home())
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

	ar.GET("/app/sync-lists", handlers.SyncListIndex)
	ar.GET("/app/sync-lists/new", handlers.SyncListNew)
	ar.POST("/app/sync-lists", handlers.SyncListCreate)
	ar.GET("/app/sync-lists/:id", handlers.SyncListShow)
	ar.GET("/app/sync-lists/:id/edit", handlers.SyncListEdit)
	ar.PUT("/app/sync-lists/:id", handlers.SyncListUpdate)
	ar.DELETE("/app/sync-lists/:id", handlers.SyncListDelete)

	ar.GET("/app/sync-lists/:id/mailboxes/new", handlers.MailboxNew)
	ar.POST("/app/sync-lists/:id/mailboxes", handlers.MailboxCreate)
	ar.DELETE("/app/sync-lists/:listId/mailboxes/:id", handlers.MailboxDelete)

	ar.POST("/app/sync-lists/:id/migrate/start", handlers.SyncListJobMigrateStart)
	ar.POST("/app/sync-lists/:id/migrate/stop", handlers.SyncListJobMigrateStop)
	ar.DELETE("/app/sync-lists/:id/migrate", handlers.SyncListDeleteJobs)
	ar.POST("/app/sync-lists/:listId/mailboxes/:id/migrate/start", handlers.MailboxJobMigrateStart)
	ar.POST("/app/sync-lists/:listId/mailboxes/:id/migrate/stop", handlers.MailboxJobMigrateStop)
	ar.DELETE("/app/sync-lists/:listId/mailboxes/:id/migrate", handlers.MailboxDeleteJob)
}
