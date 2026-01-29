package httpx

import (
	"app/errorsx"
	"errors"
	"fmt"
	"log/slog"

	"github.com/a-h/templ"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
)

func Render(c *echo.Context, statusCode int, t templ.Component) error {
	c.Response().WriteHeader(statusCode)
	return t.Render(c.Request().Context(), c.Response())
}

func RenderFragment(c *echo.Context, statusCode int, fragmentName string, t templ.Component) error {
	c.Response().WriteHeader(statusCode)
	return templ.RenderFragments(c.Request().Context(), c.Response(), t, fragmentName)
}

func Reswap(c *echo.Context, target string) {
	c.Response().Header().Set("HX-Reswap", target)
}

func Retarget(c *echo.Context, target string) {
	c.Response().Header().Set("HX-Retarget", target)
}

func Redirect(c *echo.Context, url string) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Location", url)
		return nil
	}

	return c.Redirect(303, url)
}

func FormatValues(c *echo.Context) map[string]string {
	values := make(map[string]string)
	form, err := c.FormValues()
	if err != nil {
		return values
	}

	for k, v := range form {
		if len(v) > 0 {
			values[k] = v[0]
		}
	}

	return values
}

func FormatErrors(err error) map[string]string {
	errs := make(map[string]string)

	rawErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		if errorsx.IsNotFoundError(err) {
			errs["_Error"] = MsgErrNotFound
		} else if errorsx.IsUniqueConstraintError(err) {
			errs["_Error"] = MsgErrDuplicate
		} else if errors.Is(bcrypt.ErrMismatchedHashAndPassword, err) {
			errs["_Error"] = MsgErrBadCredentials
		} else {
			slog.Error(err.Error())
			errs["_Error"] = MsgErrGeneric
		}
		return errs
	}

	for _, err := range rawErrs {
		field := err.Field()

		switch err.Tag() {
		case "required":
			errs[field] = MsgErrRequired
		case "min":
			errs[field] = fmt.Sprintf(MsgErrTooShort, err.Param())
		case "max":
			errs[field] = fmt.Sprintf(MsgErrTooLong, err.Param())
		case "eqfield":
			errs[field] = MsgErrMismatch
		default:
			errs[field] = MsgErrInvalid
		}
	}

	return errs
}
