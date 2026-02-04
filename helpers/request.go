package helpers

import (
	"strconv"

	"github.com/labstack/echo/v5"
)

func BindAndValidate(c *echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return err
	}

	if err := c.Validate(v); err != nil {
		return err
	}

	return nil
}

func ParamAsInt(c *echo.Context, param string) (int, error) {
	value := c.Param(param)
	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(value)
}

func QueryParamAsInt(c *echo.Context, param string) (int, error) {
	value := c.QueryParam(param)
	if value == "" && param == "page" {
		return 1, nil
	} else if value == "" {
		return 0, nil
	}

	return strconv.Atoi(value)
}
