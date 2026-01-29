package httpx

import "github.com/labstack/echo/v5"

func BindAndValidate(c *echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return err
	}

	if err := c.Validate(v); err != nil {
		return err
	}

	return nil
}
