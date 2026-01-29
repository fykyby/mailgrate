package handlers

import (
	"app/data"
	"app/httpx"
	"app/templates/pages/user"
	"net/http"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
)

func UserShowSignUp(c *echo.Context) error {
	return httpx.Render(c, http.StatusOK, user.SignUp(user.SignUpProps{}))
}

func UserSignUp(c *echo.Context) error {
	var req struct {
		Email           string `form:"Email" validate:"required,email,max=255"`
		Password        string `form:"Password" validate:"required,min=8,max=255"`
		PasswordConfirm string `form:"PasswordConfirm" validate:"required,min=8,max=255,eqfield=Password"`
	}

	err := httpx.BindAndValidate(c, &req)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.SignUp(user.SignUpProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.SignUp(user.SignUpProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	_, err = data.CreateUser(c.Request().Context(), req.Email, string(hashedPassword))
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.SignUp(user.SignUpProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	return httpx.Redirect(c, "/log-in")
}

func UserShowLogIn(c *echo.Context) error {
	return httpx.Render(c, http.StatusOK, user.LogIn(user.LogInProps{}))
}

func UserLogIn(c *echo.Context) error {
	var req struct {
		Email    string `form:"Email" validate:"required,email,max=255"`
		Password string `form:"Password" validate:"required,min=8,max=255"`
	}

	err := httpx.BindAndValidate(c, &req)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.LogIn(user.LogInProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	u, err := data.FindUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.LogIn(user.LogInProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password))
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.LogIn(user.LogInProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	httpx.SetUserSessionData(c, &httpx.UserSessionData{
		ID:    u.ID,
		Email: u.Email,
	})

	return httpx.Redirect(c, "/app")
}

func UserLogOut(c *echo.Context) error {
	httpx.ClearUserSessionData(c)

	return httpx.Redirect(c, "/log-in")
}
