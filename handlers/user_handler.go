package handlers

import (
	"app/config"
	"app/errorsx"
	"app/helpers"
	"app/models"
	"app/templates/components/alert"
	"app/templates/pages/base"
	"app/templates/pages/user"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
)

func UserShowSignUp(c *echo.Context) error {
	return helpers.Render(c, http.StatusOK, user.SignUp(user.SignUpProps{}))
}

func UserSignUp(c *echo.Context) error {
	var req struct {
		Email           string `form:"Email" validate:"required,email,max=255"`
		Password        string `form:"Password" validate:"required,min=8,max=255"`
		PasswordConfirm string `form:"PasswordConfirm" validate:"required,min=8,max=255,eqfield=Password"`
	}

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", user.SignUp(user.SignUpProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.SignUp(user.SignUpProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	tokenHash, token, err := helpers.GenerateToken()
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.SignUp(user.SignUpProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	confirmTokenExpiresAt := time.Now().Add(72 * time.Hour)

	u, err := models.CreateUser(c.Request().Context(), req.Email, string(hashedPassword), tokenHash, confirmTokenExpiresAt)
	if err != nil {
		slog.Error("failed to create user", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.SignUp(user.SignUpProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if config.Config.RequireEmailConfirmation {
		dialer := gomail.NewDialer(config.Config.SMTPHost, config.Config.SMTPPort, config.Config.SMTPLogin, config.Config.SMTPPassword)

		message := gomail.NewMessage()
		message.SetHeader("From", config.Config.SMTPLogin)
		message.SetHeader("To", req.Email)
		message.SetHeader("Subject", fmt.Sprintf("%s | Confirm Email", config.Config.AppName))
		message.SetBody("text/html", fmt.Sprintf("Confirm Email Link: <a href='%s'>Click here</a>", fmt.Sprintf("https://%s/sign-up/%s", c.Request().Host, token)))

		err = dialer.DialAndSend(message)
		if err != nil {
			slog.Error("failed to send email", "error", err)
			return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
				Values: helpers.FormatValues(c),
				Errors: helpers.FormatErrors(err),
			}))
		}

		return helpers.Render(c, http.StatusOK, alert.Success(helpers.MsgSuccessUserCreated))
	} else {
		u.Confirmed = true
		u.ConfirmationTokenHash = nil
		u.ConfirmationExpiresAt = nil

		_, err = models.UpdateUser(c.Request().Context(), u)
		if err != nil {
			slog.Error("failed to update user", "error", err)
			return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.SignUp(user.SignUpProps{
				Values: helpers.FormatValues(c),
				Errors: helpers.FormatErrors(err),
			}))
		}

		return helpers.Redirect(c, "/log-in")
	}
}

func UserSignUpConfirm(c *echo.Context) error {
	token := c.Param("token")
	rawTokenHash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(rawTokenHash[:])

	u, err := models.FindUserByConfirmationTokenHash(c.Request().Context(), tokenHash)
	if err != nil {
		slog.Error("failed to find user by confirmation token hash", "error", err)
		return helpers.Render(c, http.StatusNotFound, base.Error(helpers.MsgErrNotFound))
	}

	// No need to validate token expiration date - users get deleted in background goroutine if token is expired

	u.Confirmed = true
	u.ConfirmationTokenHash = nil
	u.ConfirmationExpiresAt = nil

	_, err = models.UpdateUser(c.Request().Context(), u)
	if err != nil {
		slog.Error("failed to update user", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/log-in")
}

func UserShowLogIn(c *echo.Context) error {
	return helpers.Render(c, http.StatusOK, user.LogIn(user.LogInProps{}))
}

func UserLogIn(c *echo.Context) error {
	var req struct {
		Email    string `form:"Email" validate:"required,email,max=255"`
		Password string `form:"Password" validate:"required,min=8,max=255"`
	}

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", user.LogIn(user.LogInProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	u, err := models.FindUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		slog.Error("failed to find user by email", "error", err)
		return helpers.RenderFragment(c, http.StatusNotFound, "form", user.LogIn(user.LogInProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if !u.Confirmed {
		err := errors.New("not found - user not confirmed")
		slog.Error("failed to log in user", "error", err)
		return helpers.RenderFragment(c, http.StatusNotFound, "form", user.LogIn(user.LogInProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password))
	if err != nil {
		slog.Error("failed to compare password hash", "error", err)
		return helpers.RenderFragment(c, http.StatusNotFound, "form", user.LogIn(user.LogInProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	err = helpers.SetUserSessionData(c, &helpers.UserSessionData{
		ID:    u.Id,
		Email: u.Email,
	})
	if err != nil {
		slog.Error("failed to set user session data", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.LogIn(user.LogInProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Redirect(c, "/app")
}

func UserLogOut(c *echo.Context) error {
	err := helpers.ClearUserSessionData(c)
	if err != nil {
		slog.Error("failed to clear user session data", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	return helpers.Redirect(c, "/log-in")
}

func UserShowRequestPasswordReset(c *echo.Context) error {
	values := make(map[string]string)

	u := helpers.GetUserSessionData(c)
	if u != nil {
		values["Email"] = u.Email
	}

	return helpers.Render(c, http.StatusOK, user.RequestPasswordReset(user.RequestPasswordResetProps{
		Values: values,
	}))
}

func UserRequestPasswordReset(c *echo.Context) error {
	var req struct {
		Email string `form:"Email" validate:"required,email,max=255"`
	}

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		slog.Error("failed to bind and validate request", "error", err)
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	u, err := models.FindUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusOK, alert.Success(helpers.MsgSuccessMessageSent))
		}

		slog.Error("failed to find user by email", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	tokenHash, token, err := helpers.GenerateToken()
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	u.PasswordResetTokenHash = &tokenHash
	u.PasswordResetExpiresAt = &expiresAt

	_, err = models.UpdateUser(c.Request().Context(), u)
	if err != nil {
		slog.Error("failed to update user", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	dialer := gomail.NewDialer(config.Config.SMTPHost, config.Config.SMTPPort, config.Config.SMTPLogin, config.Config.SMTPPassword)

	message := gomail.NewMessage()
	message.SetHeader("From", config.Config.SMTPLogin)
	message.SetHeader("To", u.Email)
	message.SetHeader("Subject", fmt.Sprintf("%s | Password Reset", config.Config.AppName))
	if config.Config.Debug {
		message.SetBody("text/html", fmt.Sprintf("Password Reset Link: <a href='%s'>Click Here</a>", fmt.Sprintf("http://%s/password-reset/%s", c.Request().Host, token)))
	} else {
		message.SetBody("text/html", fmt.Sprintf("Password Reset Link: <a href='%s'>Click here</a>", fmt.Sprintf("https://%s/password-reset/%s", c.Request().Host, token)))
	}

	err = dialer.DialAndSend(message)
	if err != nil {
		slog.Error("failed to send password reset email", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.RequestPasswordReset(user.RequestPasswordResetProps{
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	return helpers.Render(c, http.StatusOK, alert.Success(helpers.MsgSuccessMessageSent))
}

func UserShowPasswordReset(c *echo.Context) error {
	token := c.Param("token")
	rawTokenHash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(rawTokenHash[:])

	u, err := models.FindUserByPasswordResetTokenhash(c.Request().Context(), tokenHash)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, base.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find user by password reset token hash", "error", err)
		return helpers.Render(c, http.StatusInternalServerError, base.Error(helpers.MsgErrGeneric))
	}

	if u.Id != helpers.GetUserSessionData(c).ID {
		_ = helpers.ClearUserSessionData(c)
	}

	return helpers.Render(c, http.StatusOK, user.PasswordReset(user.PasswordResetProps{
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

	err := helpers.BindAndValidate(c, &req)
	if err != nil {
		return helpers.RenderFragment(c, http.StatusBadRequest, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	u, err := models.FindUserByPasswordResetTokenhash(c.Request().Context(), tokenHash)
	if err != nil {
		if errorsx.IsNotFoundError(err) {
			return helpers.Render(c, http.StatusNotFound, base.Error(helpers.MsgErrNotFound))
		}

		slog.Error("failed to find user by password reset token hash", "error", err)
		return helpers.RenderFragment(c, http.StatusNotFound, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if time.Now().After(*u.PasswordResetExpiresAt) {
		return helpers.RenderFragment(c, http.StatusNotFound, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	u.PasswordHash = string(hashedPassword)
	u.PasswordResetTokenHash = nil
	u.PasswordResetExpiresAt = nil

	_, err = models.UpdateUser(c.Request().Context(), u)
	if err != nil {
		slog.Error("failed to update user", "error", err)
		return helpers.RenderFragment(c, http.StatusInternalServerError, "form", user.PasswordReset(user.PasswordResetProps{
			Token:  token,
			Values: helpers.FormatValues(c),
			Errors: helpers.FormatErrors(err),
		}))
	}

	if helpers.GetUserSessionData(c) != nil {
		return helpers.Redirect(c, "/app")
	}

	return helpers.Redirect(c, "/log-in")
}
