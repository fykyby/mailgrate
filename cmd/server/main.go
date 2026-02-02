package main

import (
	efs "app"
	"app/app"
	"app/config"
	"app/db"
	"app/errorsx"
	"app/httpx"
	"app/templates/pages"
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
	if config.Config.IsDev {
		loggerOptions.Level = slog.LevelDebug
	} else {
		loggerOptions.Level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, loggerOptions))
	slog.SetDefault(logger)

	db.InitPostgresDatabase()
	defer db.Bun.Close()

	httpx.InitPostgresSessionStore()
	defer httpx.SessionStore.Close()
	defer httpx.SessionStore.StopCleanup(httpx.SessionStore.Cleanup(time.Minute * 60))

	e := echo.NewWithConfig(echo.Config{
		Filesystem: efs.StaticFS,
		Validator:  httpx.NewValidator(),
		Logger:     slog.Default(),
		Binder:     &echo.DefaultBinder{},
	})

	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		if errorsx.IsNotFoundError(err) {
			httpx.Render(c, http.StatusNotFound, pages.Error(httpx.MsgErrNotFound))
		} else {
			httpx.Render(c, http.StatusInternalServerError, pages.Error(httpx.MsgErrGeneric))
		}
	}

	e.Static("/static", "static")

	app.RegisterMiddleware(e)
	app.RegisterRoutes(e)
	app.RegisterJobs()
	app.RunBackgroundCleanUp(context.Background())

	if err := app.Start(e); err != nil {
		slog.Error(err.Error())
	}
}
