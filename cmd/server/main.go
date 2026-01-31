package main

import (
	efs "app"
	"app/app"
	"app/config"
	"app/db"
	"app/httpx"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	config.InitConfig()

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

	e.Static("/static", "static")

	app.RegisterMiddleware(e)
	app.RegisterRoutes(e)
	app.RegisterJobs()

	if err := app.Start(e); err != nil {
		slog.Error(err.Error())
	}
}
