package utils

import (
	"app/httpx"
	"context"
)

func GetUserData(ctx context.Context) *httpx.UserSessionData {
	userData, ok := ctx.Value(httpx.TemplContextSessionKey).(*httpx.UserSessionData)
	if !ok {
		return nil
	}

	return userData
}
