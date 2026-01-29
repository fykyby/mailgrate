package middleware

import (
	"app/httpx"
	"context"

	"github.com/labstack/echo/v5"
)

func WithAuthRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userSession := httpx.GetUserSessionData(c)
		if userSession == nil {
			httpx.Redirect(c, "/log-in")
			return nil
		}

		ctx := context.WithValue(c.Request().Context(), httpx.TemplContextSessionKey, userSession)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

func WithAuthForbidden(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userSession := httpx.GetUserSessionData(c)
		if userSession != nil {
			httpx.Redirect(c, "/app")
			return nil
		}

		ctx := context.WithValue(c.Request().Context(), httpx.TemplContextSessionKey, userSession)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

func WithAuthAny(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userSession := httpx.GetUserSessionData(c)

		ctx := context.WithValue(c.Request().Context(), httpx.TemplContextSessionKey, userSession)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}
