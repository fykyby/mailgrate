package handlers

import (
	"app/config"
	"app/errorsx"
	"app/httpx"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages"
	"app/templates/pages/user"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
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
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.SignUp(user.SignUpProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	_, err = models.CreateUser(c.Request().Context(), req.Email, string(hashedPassword))
	if err != nil {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.SignUp(user.SignUpProps{
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

	u, err := models.FindUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusNotFound, "form", user.LogIn(user.LogInProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password))
	if err != nil {
		return httpx.RenderFragment(c, http.StatusNotFound, "form", user.LogIn(user.LogInProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	httpx.SetUserSessionData(c, &httpx.UserSessionData{
		ID:    u.ID,
		Email: u.Email,
	})

	go models.DeletePasswordResetByUserID(context.Background(), u.ID)

	return httpx.Redirect(c, "/app")
}

func UserLogOut(c *echo.Context) error {
	httpx.ClearUserSessionData(c)

	return httpx.Redirect(c, "/log-in")
}

func UserShowRequestPasswordReset(c *echo.Context) error {
	values := make(map[string]string)

	u := httpx.GetUserSessionData(c)
	if u != nil {
		values["Email"] = u.Email
	}

	return httpx.Render(c, http.StatusOK, user.RequestPasswordReset(user.RequestPasswordResetProps{
		Values: values,
	}))
}

func UserRequestPasswordReset(c *echo.Context) error {
	var req struct {
		Email string `form:"Email" validate:"required,email,max=255"`
	}

	err := httpx.BindAndValidate(c, &req)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	u, err := models.FindUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return httpx.Render(c, http.StatusOK, alert.Success(httpx.MsgSuccessMessageSent))
		}
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	existingReset, err := models.FindPasswordResetByUserID(c.Request().Context(), u.ID)
	if err != nil && !errorsx.IsNotFoundError(err) {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	if existingReset != nil {
		err := models.DeletePasswordReset(c.Request().Context(), existingReset.ID)
		if err != nil {
			return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
				Values: httpx.FormatValues(c),
				Errors: httpx.FormatErrors(err),
			}))
		}
	}

	rawToken := make([]byte, 32)
	_, err = rand.Read(rawToken)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	token := base64.RawURLEncoding.EncodeToString(rawToken)

	rawTokenHash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(rawTokenHash[:])

	_, err = models.CreatePasswordReset(c.Request().Context(), u.ID, tokenHash, time.Now().Add(1*time.Hour))
	if err != nil {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	dialer := gomail.NewDialer(config.Config.SMTPHost, config.Config.SMTPPort, config.Config.SMTPLogin, config.Config.SMTPPassword)

	message := gomail.NewMessage()
	message.SetHeader("From", config.Config.SMTPLogin)
	message.SetHeader("To", u.Email)
	message.SetHeader("Subject", fmt.Sprintf("%s | Password Reset", config.Config.AppName))
	if config.Config.IsDev {
		message.SetBody("text/html", fmt.Sprintf("Password Reset Link: <a href='%s'>Click Here</a>", fmt.Sprintf("http://%s/password-reset/%s", c.Request().Host, token)))
	} else {
		message.SetBody("text/html", fmt.Sprintf("Password Reset Link: <a href='%s'>Click here</a>", fmt.Sprintf("https://%s/password-reset/%s", c.Request().Host, token)))
	}

	err = dialer.DialAndSend(message)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	return httpx.Render(c, http.StatusOK, alert.Success(httpx.MsgSuccessMessageSent))
}

func UserShowPasswordReset(c *echo.Context) error {
	token := c.Param("token")
	rawTokenHash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(rawTokenHash[:])

	reset, err := models.FindPasswordResetByToken(c.Request().Context(), tokenHash)
	if err != nil && !errorsx.IsNotFoundError(err) {
		return httpx.Render(c, http.StatusInternalServerError, pages.Error(httpx.MsgErrGeneric))
	}

	if reset == nil {
		return httpx.Render(c, http.StatusNotFound, pages.Error(httpx.MsgErrNotFound))
	}

	return httpx.Render(c, http.StatusOK, user.PasswordReset(user.PasswordResetProps{
		Token: token,
	}))
}

func UserPasswordReset(c *echo.Context) error {
	var req struct {
		Password        string `form:"Password" validate:"required,min=8,max=255"`
		PasswordConfirm string `form:"PasswordConfirm" validate:"required,min=8,max=255,eqfield=Password"`
	}

	token := c.Param("token")
	rawTokenHash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(rawTokenHash[:])

	err := httpx.BindAndValidate(c, &req)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	reset, err := models.FindPasswordResetByToken(c.Request().Context(), tokenHash)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusNotFound, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	u, err := models.FindUserByID(c.Request().Context(), reset.UserID)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	u.Password = string(hashedPassword)

	_, err = models.UpdateUser(c.Request().Context(), u)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusInternalServerError, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	_ = models.DeletePasswordReset(c.Request().Context(), reset.ID)

	return httpx.Redirect(c, "/log-in")
}
