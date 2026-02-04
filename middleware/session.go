package middleware

import (
	"app/helpers"
	"context"

	"github.com/labstack/echo/v5"
)

func WithAuthRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userSession := helpers.GetUserSessionData(c)
		if userSession == nil {
			co, err := c.Cookie("session")
			if err == nil {
				co.MaxAge = -1
				c.SetCookie(co)
			}
			helpers.Redirect(c, "/log-in")
			return nil
		}

		ctx := context.WithValue(c.Request().Context(), helpers.TemplContextSessionKey, userSession)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

func WithAuthForbidden(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userSession := helpers.GetUserSessionData(c)
		if userSession != nil {
			helpers.Redirect(c, "/app")
			return nil
		}

		ctx := context.WithValue(c.Request().Context(), helpers.TemplContextSessionKey, userSession)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

func WithAuthAny(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userSession := helpers.GetUserSessionData(c)

		ctx := context.WithValue(c.Request().Context(), helpers.TemplContextSessionKey, userSession)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}
