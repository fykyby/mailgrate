package app

import (
	"app/data"
	"app/handlers"
	"app/httpx"
	"app/jobs"
	"app/templates/pages"
	"encoding/json"

	middlewarex "app/middleware"

	"github.com/labstack/echo/v5"
)

func RegisterRoutes(e *echo.Echo) {
	g := e.Group("")
	g.Use(middlewarex.WithAuthAny)
	g.GET("/", func(c *echo.Context) error {
		return httpx.Render(c, 200, pages.Home())
	})

	af := e.Group("")
	af.Use(middlewarex.WithAuthForbidden)
	af.GET("/sign-up", handlers.UserShowSignUp)
	af.POST("/sign-up", handlers.UserSignUp)
	af.GET("/log-in", handlers.UserShowLogIn)
	af.POST("/log-in", handlers.UserLogIn)

	ar := e.Group("")
	ar.Use(middlewarex.WithAuthRequired)
	ar.POST("/log-out", handlers.UserLogOut)
	ar.GET("/app", func(c *echo.Context) error {
		return httpx.Render(c, 200, pages.Dashboard())
	})

	ar.GET("/job", func(c *echo.Context) error {
		jobHandler := jobs.NewExampleJob()

		payload, err := json.Marshal(jobHandler)
		if err != nil {
			return err
		}

		_, err = data.CreateJob(c.Request().Context(), httpx.GetUserSessionData(c).ID, jobs.ExampleJobType, payload)
		if err != nil {
			return err
		}

		return c.String(200, "ok")
	})
}
