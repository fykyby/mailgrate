package utils

import (
	"app/helpers"
	"context"
)

func GetUserData(ctx context.Context) *helpers.UserSessionData {
	userData, ok := ctx.Value(helpers.TemplContextSessionKey).(*helpers.UserSessionData)
	if !ok {
		return nil
	}

	return userData
}
