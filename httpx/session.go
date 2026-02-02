package httpx

import (
	"app/config"
	"encoding/gob"
	"net/http"

	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v5"
)

const sessionKey = "session"
const userSessionKey = "user"
const TemplContextSessionKey = "session"

type UserSessionData struct {
	ID    int
	Email string
}

var SessionStore *pgstore.PGStore

func InitPostgresSessionStore() {
	store, err := pgstore.NewPGStore(config.Config.DatabaseURL, []byte(config.Config.AppKey))
	if err != nil {
		panic(err)
	}

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   !config.Config.IsDev,
		SameSite: http.SameSiteLaxMode,
	}

	gob.Register(&UserSessionData{})

	SessionStore = store
}

func GetUserSessionData(c *echo.Context) *UserSessionData {
	sess, err := session.Get(sessionKey, c)
	if err != nil {
		return nil
	}

	userData := sess.Values[userSessionKey]
	if userData == nil {
		return nil
	}

	userDataValue, ok := userData.(*UserSessionData)
	if !ok {
		return nil
	}

	return userDataValue
}

func SetUserSessionData(c *echo.Context, userData *UserSessionData) error {
	sess, err := session.Get(sessionKey, c)
	if err != nil {
		return err
	}

	sess.Values[userSessionKey] = userData

	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	return nil
}

func ClearUserSessionData(c *echo.Context) error {
	sess, err := session.Get(sessionKey, c)
	if err != nil {
		return err
	}

	delete(sess.Values, userSessionKey)
	sess.Options.MaxAge = -1

	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	return nil
}
