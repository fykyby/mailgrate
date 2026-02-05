package main

import (
	efs "app"
	"app/app"
	"app/config"
	"app/db"
	"app/errorsx"
	"app/helpers"
	"app/templates/pages/base"
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	config.InitConfig()

	loggerOptions := &slog.HandlerOptions{}
	if config.Config.Debug {
		loggerOptions.Level = slog.LevelDebug
	} else {
		loggerOptions.Level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, loggerOptions))
	slog.SetDefault(logger)

	db.InitPostgresDatabase()
	defer db.Bun.Close()

	helpers.InitPostgresSessionStore()
	defer helpers.SessionStore.Close()
	defer helpers.SessionStore.StopCleanup(helpers.SessionStore.Cleanup(time.Minute * 60))

	e := echo.NewWithConfig(echo.Config{
		Filesystem: efs.StaticFS,
		Validator:  helpers.NewValidator(),
		Logger:     slog.Default(),
		Binder:     &echo.DefaultBinder{},
	})

	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		if errorsx.IsNotFoundError(err) {
			helpers.Render(c, http.StatusNotFound, base.Error(helpers.MsgErrNotFound))
		} else {
			helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
		}
	}

	e.Static("/static", "static")

	app.RegisterMiddleware(e)
	app.RegisterRoutes(e)
	app.RegisterJobs()

	if config.Config.RequireEmailConfirmation {
		app.RunBackgroundCleanUp(context.Background())
	}

	if err := app.Start(e); err != nil {
		slog.Error(err.Error())
	}
}
