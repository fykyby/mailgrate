package app

import (
	"app/handlers"
	"app/httpx"
	"app/jobs"
	"app/models"
	"app/templates/pages"
	"app/worker"
	"encoding/json"
	"net/http"
	"strconv"

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

	ar.GET("/job/new", func(c *echo.Context) error {
		payload := jobs.NewExample()

		bytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		_, err = models.CreateJob(c.Request().Context(), httpx.GetUserSessionData(c).ID, jobs.ExampleType, bytes)
		if err != nil {
			return err
		}

		return c.String(http.StatusOK, "Job started successfully")
	})

	ar.GET("/job/:id/stop", func(c *echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return err
		}

		job := worker.GetRunningJob(id)
		if job != nil {
			job.Cancel()
		}

		_, err = models.UpdateJob(c.Request().Context(), &models.Job{
			ID:     id,
			Status: models.JobStatusInterrupted,
		})
		if err != nil {
			return err
		}

		return c.String(http.StatusOK, "Job interrupted successfully")
	})

	ar.GET("/job/:id/start", func(c *echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return err
		}

		_, err = models.UpdateJob(c.Request().Context(), &models.Job{
			ID:     id,
			Status: models.JobStatusPending,
		})
		if err != nil {
			return err
		}

		return c.String(http.StatusOK, "Job unpaused successfully")
	})
}
