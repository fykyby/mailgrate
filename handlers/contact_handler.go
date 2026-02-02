package handlers

import (
	"app/config"
	"app/httpx"
	"app/templates/components/alert"
	"app/templates/pages"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"gopkg.in/gomail.v2"
)

func ContactShow(c *echo.Context) error {
	return httpx.Render(c, http.StatusOK, pages.Contact(pages.ContactProps{}))
}

func ContactSend(c *echo.Context) error {
	var req struct {
		Email   string `form:"Email" validate:"required,email,max=255"`
		Message string `form:"Message" validate:"required,min=16,max=2048"`
	}

	err := httpx.BindAndValidate(c, &req)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", pages.Contact(pages.ContactProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	dialer := gomail.NewDialer(config.Config.SMTPHost, config.Config.SMTPPort, config.Config.SMTPLogin, config.Config.SMTPPassword)

	message := gomail.NewMessage()
	message.SetHeader("From", config.Config.SMTPLogin)
	message.SetHeader("To", config.Config.SMTPLogin)
	message.SetHeader("Reply-To", req.Email)
	message.SetHeader("Subject", fmt.Sprintf("%s | Contact Form Submission | %s", req.Email, config.Config.AppName))
	message.SetBody("text/html", fmt.Sprintf("Email: %s<br>Message: %s", req.Email, req.Message))

	err = dialer.DialAndSend(message)
	if err != nil {
		return httpx.RenderFragment(c, http.StatusBadRequest, "form", pages.Contact(pages.ContactProps{
			Values: httpx.FormatValues(c),
			Errors: httpx.FormatErrors(err),
		}))
	}

	return httpx.Render(c, http.StatusOK, alert.Success(httpx.MsgSuccessMessageSent))
}
